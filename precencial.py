import csv
import os
import platform
import tkinter as tk
from datetime import datetime
from pathlib import Path
from tkinter import simpledialog
from typing import Optional


# Funções de interface personalizada
def show_custom_message(title: str, message: str, kind: str = "info") -> None:
    window = tk.Toplevel()
    window.title(title)
    window.resizable(False, False)
    window.attributes("-topmost", True)
    window.geometry("+%d+%d" % (
        window.winfo_screenwidth() // 2 - 150,
        window.winfo_screenheight() // 2 - 75
    ))

    tk.Label(window, text=message, padx=20, pady=20, wraplength=300).pack()
    btn_text = "OK" if kind != "error" else "Fechar"
    tk.Button(window, text=btn_text, command=window.destroy, width=10).pack(pady=10)
    window.grab_set()
    window.wait_window()


def ask_yes_no(title: str, question: str) -> bool:
    result = [False]  # Default result

    def on_yes():
        result[0] = True
        window.destroy()

    def on_no():
        result[0] = False
        window.destroy()

    window = tk.Toplevel()
    window.title(title)
    window.resizable(False, False)
    window.attributes("-topmost", True)
    window.geometry("+%d+%d" % (
        window.winfo_screenwidth() // 2 - 150,
        window.winfo_screenheight() // 2 - 75
    ))

    tk.Label(window, text=question, padx=20, pady=20, wraplength=300).pack()
    frame = tk.Frame(window)
    frame.pack(pady=10)
    tk.Button(frame, text="Sim", command=on_yes, width=10).pack(side="left", padx=5)
    tk.Button(frame, text="Não", command=on_no, width=10).pack(side="right", padx=5)

    window.grab_set()
    window.wait_window()
    return result[0]


def ask_area_selection(title: str, prompt: str, options: list[str]) -> Optional[str]:
    result = [None]

    def on_select():
        selected = var.get()
        if selected:
            result[0] = selected
            window.destroy()
        else:
            error_label.config(text="Selecione uma área.", fg="red")

    window = tk.Toplevel()
    window.title(title)
    window.resizable(False, False)
    window.attributes("-topmost", True)
    window.geometry("+%d+%d" % (
        window.winfo_screenwidth() // 2 - 150,
        window.winfo_screenheight() // 2 - 100
    ))

    tk.Label(window, text=prompt, padx=20, pady=10).pack()
    var = tk.StringVar()
    for opt in options:
        tk.Radiobutton(window, text=opt, variable=var, value=opt).pack(anchor="w", padx=20)

    error_label = tk.Label(window, text="")
    error_label.pack()

    tk.Button(window, text="Confirmar", command=on_select, width=15).pack(pady=10)
    window.grab_set()
    window.wait_window()
    return result[0]


# Classe principal
class PresenceTracker:
    FOLDER_NAME = "Precencial"
    CSV_FILENAME = "registros.csv"
    CONFIG_FILENAME = "config.txt"
    CSV_HEADERS = ["data", "hora", "resposta", "observacao", "area"]

    YES = "Sim"
    NO = "Não"
    EXTRA = "extra"

    AREAS = ["AG", "CT", "CEIC", "OUTRO"]

    def __init__(self):
        self.data_folder = self._initialize_data_folder()
        self.csv_path = self.data_folder / self.CSV_FILENAME
        self.config_path = self.data_folder / self.CONFIG_FILENAME
        self.monthly_goal = self._load_or_setup_config()
        self._ensure_csv_exists()

    def _initialize_data_folder(self) -> Path:
        base_path = Path(os.environ["USERPROFILE"] if platform.system() == "Windows"
                         else os.path.expanduser("~"))
        folder_path = base_path / self.FOLDER_NAME
        folder_path.mkdir(exist_ok=True)
        return folder_path

    def _ensure_csv_exists(self) -> None:
        if not self.csv_path.exists():
            with open(self.csv_path, mode='w', newline='') as file:
                writer = csv.writer(file)
                writer.writerow(self.CSV_HEADERS)

    def _load_or_setup_config(self) -> int:
        if self.config_path.exists():
            with open(self.config_path, "r") as file:
                try:
                    return int(file.read().strip())
                except ValueError:
                    return 8

        goal = simpledialog.askinteger(
            "Configuração Inicial",
            "Quantos dias presenciais por mês você deseja atingir?",
            minvalue=1, maxvalue=31
        )
        if not goal:
            show_custom_message("Erro", "Valor inválido. Usando valor padrão de 8.", kind="error")
            goal = 8

        with open(self.config_path, "w") as file:
            file.write(str(goal))
        return goal

    def contar_presencas_mes(self) -> tuple[int, list[str]]:
        atual = datetime.now()
        total = 0
        registros = []
        try:
            with open(self.csv_path, mode='r') as file:
                reader = csv.DictReader(file)
                for row in reader:
                    data = datetime.strptime(row["data"], "%Y-%m-%d")
                    if data.year == atual.year and data.month == atual.month and row["resposta"] == self.YES:
                        total += 1
                        area = row.get('area') or 'N/A'
                        registros.append(f"* {row['data']} Presencial no\t{area}")
            return total, registros
        except (IOError, csv.Error) as e:
            show_custom_message("Erro", f"Falha ao ler os registros: {str(e)}", kind="error")
            return 0, []

    def salvar_presenca(self, presente: bool, observacao: str = "", area: str = "") -> None:
        agora = datetime.now()
        resposta = self.YES if presente else self.NO
        try:
            with open(self.csv_path, mode='a', newline='') as file:
                writer = csv.writer(file)
                writer.writerow([
                    agora.date(),
                    agora.strftime("%H:%M:%S"),
                    resposta,
                    observacao,
                    area
                ])
            show_custom_message("Registrado", "Sua resposta foi salva com sucesso.")
        except IOError as e:
            show_custom_message("Erro", f"Falha ao salvar a presença: {str(e)}", kind="error")

    def perguntar_presenca(self) -> None:
        total, registros = self.contar_presencas_mes()

        resumo = f"Você já marcou presença {total} vez(es) neste mês."
        if registros:
            resumo += "\n\n" + "\n".join(registros)

        show_custom_message("Resumo do mês", resumo)

        presente = ask_yes_no("Precencial", "Você está presencial hoje?")

        if not presente:
            self.salvar_presenca(False)
            show_custom_message("Informativo", "Tudo bem. Hoje não será contado como presencial.")
            return

        if total >= self.monthly_goal:
            if not ask_yes_no(
                    "Meta Atingida",
                    f"Você já registrou {self.monthly_goal} vezes este mês.\nDeseja registrar mais uma por conta própria?"
            ):
                show_custom_message("Encerrado", "Nenhum registro foi adicionado.")
                return
            observacao = self.EXTRA
        else:
            observacao = ""

        area = ask_area_selection("\u00c1rea", f"Em qual área você está hoje?", self.AREAS)

        if not area:
            show_custom_message("Erro", "Área inválida. Registro cancelado.", kind="error")
            return

        self.salvar_presenca(True, observacao, area)


if __name__ == "__main__":
    root = None
    try:
        root = tk.Tk()
        root.withdraw()
        app = PresenceTracker()
        app.perguntar_presenca()
    except Exception as e:
        show_custom_message("Erro", f"Ocorreu um erro: {str(e)}", kind="error")
    finally:
        if root and root.winfo_exists():
            root.destroy()