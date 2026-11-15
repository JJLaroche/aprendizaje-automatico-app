import os
import json
from github import Github
from google import genai
from langchain_chroma import Chroma
from langchain_google_genai import GoogleGenerativeAIEmbeddings

# 1. Variables de entorno
github_token = os.environ["GITHUB_TOKEN"]
gemini_api_key = os.environ["GEMINI_API_KEY"]
pr_number = int(os.environ["PR_NUMBER"])
repo_name = os.environ["GITHUB_REPOSITORY"]

# 2. Obtener diff del PR
g = Github(github_token)
repo = g.get_repo(repo_name)
pr = repo.get_pull(pr_number)

files_changed = pr.get_files()
diff_text = ""
changed_files_list = []

for file in files_changed:
    if file.patch:
        diff_text += f"\n--- Archivo: {file.filename} ---\n{file.patch}\n"
        changed_files_list.append(file.filename)

if not diff_text.strip():
    print("No se encontraron cambios de texto.")
    exit(0)

# 3. RAG Retrieval: Buscar las reglas aplicables en ChromaDB
embeddings = GoogleGenerativeAIEmbeddings(
    model="models/gemini-embedding-001", 
    google_api_key=gemini_api_key
)

vector_store = Chroma(
    persist_directory="./chroma_db", 
    embedding_function=embeddings
)

query = f"Reglas de desarrollo, arquitectura y seguridad para los archivos: {', '.join(changed_files_list)}"
relevant_docs = vector_store.similarity_search(query, k=5)

rag_context = "\n\n".join([
    f"--- Regla/Guía ({doc.metadata.get('source', '')}) ---\n{doc.page_content}" 
    for doc in relevant_docs
])

# 4. Invocación a Gemini 3.8 Flash con Contexto RAG
prompt = f"""
REGLAS Y ESTÁNDARES INTERNOS DE LA ORGANIZACIÓN (Recuperados vía RAG):
--------------------------------------------------------------
{rag_context}
--------------------------------------------------------------

PULL REQUEST A EVALUAR:
Título: {pr.title}
Descripción: {pr.body or 'Sin descripción'}

DIFF:
{diff_text}
"""

system_instruction = """
Eres un Senior Code Reviewer. Tu objetivo es evaluar el Pull Request verificando rigurosamente la adhesión a las REGLAS Y ESTÁNDARES INTERNOS provistos.

Devuelve EXCLUSIVAMENTE un objeto JSON válido con este formato:
{
  "score_total": 18,
  "verdict": "APPROVE" | "NEEDS_CHANGES" | "COMMENT",
  "summary": "Resumen explicando el cumplimiento o incumplimiento de los estándares internos.",
  "comments": [
    {
      "path": "ruta/al/archivo.ext",
      "line": 10,
      "rule_violated": "Nombre o extracto de la regla violada (si aplica)",
      "type": "SECURITY" | "STYLE" | "ARCHITECTURE",
      "comment": "Explicación del hallazgo.",
      "suggestion": "Código sugerido (opcional)."
    }
  ]
}
"""

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

# 5. Publicar comentario en GitHub
comment_body = f"## 🤖 RAG LLM Judge Code Review\n\n" \
               f"**Veredicto:** `{evaluation['verdict']}` | **Puntuación:** {evaluation['score_total']}/20\n\n" \
               f"### Resumen:\n{evaluation['summary']}\n\n"

if evaluation.get("comments"):
    comment_body += "### Observaciones:\n"
    for item in evaluation["comments"]:
        comment_body += f"* **`{item['path']}`** (Línea {item.get('line', 'N/A')})\n"
        if item.get("rule_violated"):
            comment_body += f"  * ⚠️ **Regla:** {item['rule_violated']}\n"
        comment_body += f"  * **Comentario:** {item['comment']}\n"
        if item.get("suggestion"):
            comment_body += f"  ```suggestion\n  {item['suggestion']}\n  ```\n"

pr.create_issue_comment(comment_body)