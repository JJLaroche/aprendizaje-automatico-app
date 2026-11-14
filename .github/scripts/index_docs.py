import glob
import os
from langchain_community.document_loaders import UnstructuredMarkdownLoader
from langchain_text_splitters import MarkdownHeaderTextSplitter
from langchain_chroma import Chroma
from langchain_google_genai import GoogleGenerativeAIEmbeddings

gemini_api_key = os.environ["GEMINI_API_KEY"]

# 1. Leer los archivos Markdown de la carpeta docs/
files = glob.glob("docs/**/*.md", recursive=True)
documents = []

headers_to_split_on = [
    ("#", "Header 1"),
    ("##", "Header 2"),
    ("###", "Header 3"),
]
markdown_splitter = MarkdownHeaderTextSplitter(headers_to_split_on=headers_to_split_on)

for file in files:
    with open(file, "r", encoding="utf-8") as f:
        content = f.read()
        splits = markdown_splitter.split_text(content)
        for split in splits:
            split.metadata["source"] = file
            documents.append(split)

if not documents:
    print("No se encontraron documentos para indexar.")
    exit(0)

# 2. Generar embeddings e indexar en ChromaDB local
embeddings = GoogleGenerativeAIEmbeddings(
    model="models/text-embedding-004",
    google_api_key=gemini_api_key
)

vector_store = Chroma.from_documents(
    documents=documents,
    embedding=embeddings,
    persist_directory="./chroma_db"
)

print(f"✅ Indexados {len(documents)} fragmentos de documentación en ./chroma_db")