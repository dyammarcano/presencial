import csv
import os
import platform
from datetime import datetime
import tkinter as tk
from tkinter import messagebox

# Definir caminho da pasta "Precencial" no diretório do usuário
def obter_pasta_precencial():
    if platform.system() == "Windows":
        return os.path.join(os.environ["USERPROFILE"], "Precencial")
    else:
        return os.path.join(os.path.expanduser("~"), "Precencial")

pasta_usuario = obter_pasta_precencial()
os.makedirs(pasta_usuario, exist_ok=True)

arquivo_csv = os.path.join(pasta_usuario, "registros.csv")
meta_mensal = 8

# Criar CSV se não existir
if not os.path.exists(arquivo_csv):
    with open(arquivo_csv, mode='w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(["data", "hora", "resposta", "observacao"])

# Contar "Sim" do mês atual
def contar_sim_mes():
    agora = datetime.now()
    total = 0
    with open(arquivo_csv, mode='r') as file:
        reader = csv.DictReader(file)
        for row in reader:
            data = datetime.strptime(row["data"], "%Y-%m-%d")
            if data.year == agora.year and data.month == agora.month and row["resposta"] == "Sim":
                total += 1
    return total

# Salvar resposta no CSV
def salvar_resposta(resposta, observacao=""):
    agora = datetime.now()
    with open(arquivo_csv, mode='a', newline='') as file:
        writer = csv.writer(file)
        writer.writerow([agora.date(), agora.strftime("%H:%M:%S"), resposta, observacao])
    messagebox.showinfo("Registrado", "Sua resposta foi registrada com sucesso.")

# Lógica da pergunta
def perguntar():
    total_sim = contar_sim_mes()

    if total_sim >= meta_mensal:
        continuar = messagebox.askyesno(
            "Meta Atingida",
            f"Você já registrou {meta_mensal} vezes este mês.\nDeseja registrar mais uma por conta própria?"
        )
        if continuar:
            salvar_resposta("Sim", "extra")
        else:
            messagebox.showinfo("Tudo certo", "Nenhum registro foi adicionado.")
        return

    resposta = messagebox.askyesno("Precencial", "Você esta precencial hoje?")
    if resposta:
        salvar_resposta("Sim")
    else:
        salvar_resposta("Não")

# Iniciar sem janela principal
root = tk.Tk()
root.withdraw()
perguntar()
