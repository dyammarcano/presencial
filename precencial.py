import csv
import logging
from dataclasses import dataclass
from datetime import datetime
from enum import Enum
from pathlib import Path
from tkinter import Tk, Toplevel, Label, Button, simpledialog, StringVar, Frame
from tkinter.ttk import Radiobutton
from typing import Tuple, List, Optional

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    filename='presence_tracker.log'
)


class ResponseType(Enum):
    YES = "Sim"
    NO = "Não"


class Area(Enum):
    AG = "AG"
    CT = "CT"
    CEIC = "CEIC"
    OTHER = "OUTRO"


@dataclass
class AppConfig:
    FOLDER_NAME: str = "Precencial"
    CSV_FILENAME: str = "registros.csv"
    CONFIG_FILENAME: str = "config.txt"
    CSV_HEADERS: List[str] = ("data", "hora", "resposta", "observacao", "area")
    DEFAULT_GOAL: int = 8
    MAX_GOAL: int = 31
    EXTRA_LABEL: str = "extra"


class DialogWindow:
    def __init__(self):
        self.window = None

    def create_window(self, title: str) -> None:
        self.window = Toplevel()
        self.window.title(title)
        self.window.resizable(False, False)
        self.window.attributes("-topmost", True)
        self.center_window()

    def center_window(self, width: int = 300, height: int = 150) -> None:
        screen_width = self.window.winfo_screenwidth()
        screen_height = self.window.winfo_screenheight()
        x = (screen_width - width) // 2
        y = (screen_height - height) // 2
        self.window.geometry(f"+{x}+{y}")

    def destroy(self) -> None:
        if self.window:
            self.window.destroy()


class MessageDialog(DialogWindow):
    def show(self, title: str, message: str, kind: str = "info") -> None:
        try:
            self.create_window(title)
            Label(self.window, text=message, padx=20, pady=20, wraplength=350, font=("Courier", 12)).pack()
            btn_text = "OK" if kind != "error" else "Fechar"
            Button(self.window, text=btn_text, command=self.window.destroy, width=10).pack(pady=10)
            self.window.grab_set()
            self.window.wait_window()
        except Exception as e:
            logging.error(f"Error showing message dialog: {str(e)}")
            raise


class PresenceManager:
    def __init__(self, config: AppConfig):
        self.config = config
        self.message_dialog = MessageDialog()
        self.data_folder = self._initialize_data_folder()
        self.csv_path = self.data_folder / config.CSV_FILENAME
        self.config_path = self.data_folder / config.CONFIG_FILENAME
        self.monthly_goal = self._load_or_setup_config()
        self._ensure_csv_exists()

    def _initialize_data_folder(self) -> Path:
        try:
            home = Path.home()
            folder_path = home / self.config.FOLDER_NAME
            folder_path.mkdir(exist_ok=True)
            return folder_path
        except Exception as e:
            logging.error(f"Failed to initialize data folder: {str(e)}")
            raise

    def _ensure_csv_exists(self) -> None:
        try:
            if not self.csv_path.exists():
                with open(self.csv_path, mode='w', newline='', encoding='utf-8') as file:
                    writer = csv.writer(file)
                    writer.writerow(self.config.CSV_HEADERS)
        except Exception as e:
            logging.error(f"Failed to create CSV file: {str(e)}")
            raise

    def save_presence(self, is_present: bool, observation: str = "", area: str = "") -> None:
        try:
            now = datetime.now()
            response = ResponseType.YES.value if is_present else ResponseType.NO.value
            with open(self.csv_path, mode='a', newline='', encoding='utf-8') as file:
                writer = csv.writer(file)
                writer.writerow([
                    now.date(),
                    now.strftime("%H:%M:%S"),
                    response,
                    observation,
                    area
                ])
        except Exception as e:
            logging.error(f"Failed to save presence: {str(e)}")
            raise

    def count_monthly_presence(self) -> Tuple[int, List[str]]:
        try:
            current = datetime.now()
            total = 0
            records = []

            with open(self.csv_path, mode='r', encoding='utf-8') as file:
                reader = csv.DictReader(file)
                for row in reader:
                    date = datetime.strptime(row["data"], "%Y-%m-%d")
                    if (date.year == current.year and
                            date.month == current.month and
                            row["resposta"] == ResponseType.YES.value):
                        total += 1
                        area = row.get('area') or 'N/A'
                        records.append(f"* {row['data']} Presencial no {area.ljust(6)}")

            return total, records
        except Exception as e:
            logging.error(f"Failed to count monthly presence: {str(e)}")
            return 0, []

    def _load_or_setup_config(self) -> int:
        try:
            if self.config_path.exists():
                with open(self.config_path, "r", encoding='utf-8') as file:
                    try:
                        value = int(file.read().strip())
                        if 1 <= value <= self.config.MAX_GOAL:
                            return value
                    except ValueError:
                        logging.warning("Invalid config value found, using default")
                        return self.config.DEFAULT_GOAL

            goal = simpledialog.askinteger(
                "Configuração Inicial",
                "Quantos dias presenciais por mês você deseja atingir?",
                minvalue=1,
                maxvalue=self.config.MAX_GOAL
            )

            if not goal:
                logging.info("Using default goal value")
                goal = self.config.DEFAULT_GOAL
                self.message_dialog.show(
                    "Erro",
                    f"Valor inválido. Usando valor padrão de {self.config.DEFAULT_GOAL}.",
                    "error"
                )

            with open(self.config_path, "w", encoding='utf-8') as file:
                file.write(str(goal))

            return goal

        except Exception as e:
            logging.error(f"Error loading/setting up config: {str(e)}")
            return self.config.DEFAULT_GOAL


def _ask_area_selection() -> Optional[str]:
    dialog = DialogWindow()
    result = [None]
    var = StringVar()

    def on_select():
        selected = var.get()
        if selected:
            result[0] = selected
            dialog.destroy()
        else:
            error_label.config(text="Selecione uma área.", fg="red")

    try:
        dialog.create_window("Área")
        Label(dialog.window, text="Em qual área você está hoje?",
              padx=20, pady=10, font=("Courier", 12)).pack()

        for area in Area:
            Radiobutton(dialog.window, text=area.value, variable=var,
                        value=area.value).pack(anchor="w", padx=20)

        error_label = Label(dialog.window, text="", font=("Courier", 12))
        error_label.pack()

        Button(dialog.window, text="Confirmar", command=on_select,
               width=15).pack(pady=10)

        dialog.window.grab_set()
        dialog.window.wait_window()
        return result[0]
    except Exception as e:
        logging.error(f"Error in area selection dialog: {str(e)}")
        return None


class PresenceUI:
    def __init__(self, manager: PresenceManager):
        self.manager = manager
        self.message_dialog = MessageDialog()

    def _show_summary(self, total: int, records: List[str]) -> None:
        summary = f"Você já marcou presença {total} vez(es) neste mês."
        if records:
            summary += "\n\n" + "\n".join(records)
        self.message_dialog.show("Resumo do mês", summary)

    def _ask_is_present(self) -> bool:
        dialog = DialogWindow()
        result = [False]

        def on_yes():
            result[0] = True
            dialog.destroy()

        def on_no():
            result[0] = False
            dialog.destroy()

        try:
            dialog.create_window("Precencial")
            Label(dialog.window, text="Você está presencial hoje?",
                  padx=20, pady=20, wraplength=350, font=("Courier", 12)).pack()

            frame = Frame(dialog.window)
            frame.pack(pady=10)

            Button(frame, text="Sim", command=on_yes, width=10).pack(side="left", padx=5)
            Button(frame, text="Não", command=on_no, width=10).pack(side="right", padx=5)

            dialog.window.grab_set()
            dialog.window.wait_window()
            return result[0]
        except Exception as e:
            logging.error(f"Error in presence question dialog: {str(e)}")
            return False

    def _handle_present_response(self, total: int) -> None:
        if total >= self.manager.monthly_goal:
            if not self._ask_extra_confirmation():
                self.message_dialog.show("Encerrado", "Nenhum registro foi adicionado.")
                return
            observation = self.manager.config.EXTRA_LABEL
        else:
            observation = ""

        area = _ask_area_selection()
        if not area:
            self.message_dialog.show("Erro", "Área inválida. Registro cancelado.", "error")
            return

        self.manager.save_presence(True, observation, area)
        self.message_dialog.show("Registrado", "Sua resposta foi salva com sucesso.")

    def _handle_absent_response(self) -> None:
        self.manager.save_presence(False)
        self.message_dialog.show("Informativo",
                                 "Tudo bem. Hoje não será contado como presencial.")

    def _ask_extra_confirmation(self) -> bool:
        return self._ask_yes_no(
            "Meta Atingida",
            f"Você já registrou {self.manager.monthly_goal} vezes este mês.\n"
            f"Deseja registrar mais uma por conta própria?"
        )

    def _ask_yes_no(self, title: str, question: str) -> bool:
        dialog = DialogWindow()
        result = [False]

        def on_yes():
            result[0] = True
            dialog.destroy()

        def on_no():
            result[0] = False
            dialog.destroy()

        try:
            dialog.create_window(title)
            Label(dialog.window, text=question, padx=20, pady=20, wraplength=350, font=("Courier", 12)).pack()

            frame = Frame(dialog.window)
            frame.pack(pady=10)

            Button(frame, text="Sim", command=on_yes, width=10).pack(side="left", padx=5)
            Button(frame, text="Não", command=on_no, width=10).pack(side="right", padx=5)

            dialog.window.grab_set()
            dialog.window.wait_window()
            return result[0]
        except Exception as e:
            logging.error(f"Error in yes/no dialog: {str(e)}")
            return False

    def ask_presence(self) -> None:
        try:
            total, records = self.manager.count_monthly_presence()
            self._show_summary(total, records)

            if self._ask_is_present():
                self._handle_present_response(total)
            else:
                self._handle_absent_response()
        except Exception as e:
            logging.error(f"Error in presence dialog: {str(e)}")
            self.message_dialog.show("Erro", str(e), "error")


def main():
    root = None
    try:
        root = Tk()
        root.withdraw()

        config = AppConfig()
        manager = PresenceManager(config)
        ui = PresenceUI(manager)
        ui.ask_presence()
    except Exception as e:
        logging.error(f"Application error: {str(e)}")
        MessageDialog().show("Erro", f"Ocorreu um erro: {str(e)}", "error")
    finally:
        if root and root.winfo_exists():
            root.destroy()


if __name__ == "__main__":
    main()
