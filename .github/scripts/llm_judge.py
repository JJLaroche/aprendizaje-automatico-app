import os
import json
from github import Github
from google import genai

# 1. Obtener variables del entorno inyectadas por la Action
github_token = os.environ["GITHUB_TOKEN"]
gemini_api_key = os.environ["GEMINI_API_KEY"]
pr_number = int(os.environ["PR_NUMBER"])
repo_name = os.environ["GITHUB_REPOSITORY"]

# 2. Iniciar cliente de GitHub y obtener el diff del PR
g = Github(github_token)
repo = g.get_repo(repo_name)
pr = repo.get_pull(pr_number)

files_changed = pr.get_files()
diff_text = ""

for file in files_changed:
    if file.patch:
        diff_text += f"\n--- Archivo: {file.filename} ---\n{file.patch}\n"

# Si no hay diffs legibles (ej. solo cambio de binarios), finalizamos
if not diff_text.strip():
    print("No se encontraron cambios de texto para evaluar.")
    exit(0)

# 3. Armar el prompt con el diff
prompt = f"""
Evalúa el siguiente diff de Pull Request:

Título del PR: {pr.title}
Descripción: {pr.body or 'Sin descripción'}

Diff completo:
{diff_text}
"""

system_instruction = """
Eres un Senior Code Reviewer experto. Tu objetivo es evaluar el Pull Request en base a correctitud, seguridad, legibilidad y mantenibilidad.

Devuelve EXCLUSIVAMENTE un objeto JSON válido con este formato:
{
  "score_total": 18,
  "verdict": "APPROVE" | "NEEDS_CHANGES" | "COMMENT",
  "summary": "Resumen ejecutivo claro sobre la calidad del PR.",
  "comments": [
    {
      "path": "ruta/al/archivo.ext",
      "line": 10,
      "type": "SECURITY" | "BUG" | "STYLE",
      "comment": "Explicación detallada del hallazgo.",
      "suggestion": "Código sugerido para reemplazar (opcional)."
    }
  ]
}
"""

# 4. Invocación a Gemini 2.5 Flash con salida JSON estructurada
client = genai.Client(api_key=gemini_api_key)
response = client.models.generate_content(
    model="gemini-3.8-flash",
    contents=prompt,
    config={
        "system_instruction": system_instruction,
        "response_mime_type": "application/json"
    }
)

evaluation = json.loads(response.text)

# 5. Formatear y publicar el comentario en el PR
comment_body = f"## 🤖 LLM Judge Code Review\n\n" \
               f"**Veredicto:** `{evaluation['verdict']}` | **Puntuación:** {evaluation['score_total']}/20\n\n" \
               f"### Resumen:\n{evaluation['summary']}\n\n"

if evaluation.get("comments"):
    comment_body += "### Observaciones:\n"
    for item in evaluation["comments"]:
        comment_body += f"* **`{item['path']}`** (Línea {item.get('line', 'N/A')}) - *[{item.get('type', 'GENERAL')}]*\n"
        comment_body += f"  {item['comment']}\n"
        if item.get("suggestion"):
            comment_body += f"  ```suggestion\n  {item['suggestion']}\n  ```\n"

pr.create_issue_comment(comment_body)