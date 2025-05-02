import csv
import logging
from dataclasses import dataclass
from datetime import datetime
from enum import Enum
from pathlib import Path
from tkinter import Tk, Toplevel, Label, Button, simpledialog, Frame
from tkinter.ttk import Radiobutton
from typing import Tuple, List, Callable

# Logging configuration
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    filename='presence_tracker.log'
)


# Configuration for UI appearance
@dataclass
class UIConfig:
    FONT_FAMILY: str = "Courier"
    FONT_SIZE: int = 12
    BUTTON_WIDTH: int = 10
    DIALOG_PADDING: int = 20
    WRAP_LENGTH: int = 350


# Configuration for app data and thresholds
@dataclass
class AppConfig:
    FOLDER_NAME: str = "Precencial"
    CSV_FILENAME: str = "registros.csv"
    CONFIG_FILENAME: str = "config.txt"
    CSV_HEADERS: Tuple[str, ...] = ("data", "hora", "resposta", "observacao", "area")
    DEFAULT_GOAL: int = 8
    MAX_GOAL: int = 31
    EXTRA_LABEL: str = "extra"


# Possible responses
class ResponseType(Enum):
    YES = "Sim"
    NO = "Não"


# Work areas
class Area(Enum):
    AG = "AG"
    CT = "CT"
    CEIC = "CEIC"
    OTHER = "OUTRO"


# Base class for all dialogs
class BaseDialog:
    def __init__(self):
        self.window = None
        self.ui_config = UIConfig()

    def create_window(self, title: str) -> None:
        # Creates the main dialog window and centers it
        self.window = Toplevel()
        self.window.title(title)
        self.window.resizable(False, False)
        self.window.attributes("-topmost", True)
        self._center_window()

    def _center_window(self) -> None:
        # Centers the dialog window on the screen
        screen_width = self.window.winfo_screenwidth()
        screen_height = self.window.winfo_screenheight()
        x = (screen_width - 300) // 2
        y = (screen_height - 150) // 2
        self.window.geometry(f"+{x}+{y}")

    def create_label(self, text: str) -> Label:
        # Creates a formatted label
        return Label(
            self.window,
            text=text,
            padx=self.ui_config.DIALOG_PADDING,
            pady=self.ui_config.DIALOG_PADDING,
            wraplength=self.ui_config.WRAP_LENGTH,
            font=(self.ui_config.FONT_FAMILY, self.ui_config.FONT_SIZE)
        )

    def create_button(self, text: str, command: Callable[[], None]) -> Button:
        # Creates a button with the configured appearance
        return Button(
            self.window,
            text=text,
            command=command,
            width=self.ui_config.BUTTON_WIDTH,
            font=(self.ui_config.FONT_FAMILY, self.ui_config.FONT_SIZE)
        )

    def destroy(self) -> None:
        # Destroys the window if open
        if self.window:
            self.window.destroy()


# Message dialog built on top of BaseDialog
class MessageDialog(BaseDialog):
    def show(self, title: str, message: str, kind: str = "info") -> None:
        try:
            self.create_window(title)
            self.create_label(message).pack()
            btn_text = "OK" if kind != "error" else "Fechar"
            self.create_button(btn_text, self.window.destroy).pack(pady=10)
            self.window.grab_set()
            self.window.wait_window()
        except Exception as e:
            logging.error(f"Error showing message dialog: {str(e)}")
            raise


# Presence manager handles file I/O and config
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
        # Ensure the data folder exists in the user home directory
        home = Path.home()
        folder_path = home / self.config.FOLDER_NAME
        folder_path.mkdir(exist_ok=True)
        return folder_path

    def _ensure_csv_exists(self) -> None:
        # Create the CSV file with headers if it doesn't exist
        if not self.csv_path.exists():
            with open(self.csv_path, mode='w', newline='', encoding='utf-8') as file:
                writer = csv.writer(file)
                writer.writerow(self.config.CSV_HEADERS)

    def save_presence(self, is_present: bool, observation: str = "", area: str = "") -> None:
        # Append a new presence record to the CSV file
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

    def count_monthly_presence(self) -> Tuple[int, List[str]]:
        # Count the number of "Sim" responses in the current month
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

    def _load_or_setup_config(self) -> int:
        # Load goal config from file or prompt user if not present
        try:
            if self.config_path.exists():
                with open(self.config_path, "r", encoding='utf-8') as file:
                    value = int(file.read().strip())
                    if 1 <= value <= self.config.MAX_GOAL:
                        return value
            # Prompt user for goal if file doesn't exist or is invalid
            goal = simpledialog.askinteger(
                "Configuração Inicial",
                "Quantos dias presenciais por mês você deseja atingir?",
                minvalue=1,
                maxvalue=self.config.MAX_GOAL
            )
            if not goal:
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


# UI class handling user interaction and flow
class PresenceUI:
    def __init__(self, manager: PresenceManager):
        self.manager = manager
        self.message_dialog = MessageDialog()

    def _show_summary(self, total: int, records: List[str]) -> None:
        # Display summary of presence records for the current month
        summary = f"Você já marcou presença {total} vez(es) neste mês."
        if records:
            summary += "\n\n" + "\n".join(records)
        self.message_dialog.show("Resumo do mês", summary)

    def _ask_is_present(self) -> bool:
        # Ask user if they are present today
        dialog = BaseDialog()
        result = [False]

        def on_yes():
            result[0] = True
            dialog.destroy()

        def on_no():
            result[0] = False
            dialog.destroy()

        dialog.create_window("Precencial")
        dialog.create_label("Você está presencial hoje?").pack()

        frame = Frame(dialog.window)
        frame.pack(pady=10)
        dialog.create_button("Sim", on_yes).pack(in_=frame, side="left", padx=5)
        dialog.create_button("Não", on_no).pack(in_=frame, side="right", padx=5)

        dialog.window.grab_set()
        dialog.window.wait_window()
        return result[0]

    def _ask_extra_confirmation(self) -> bool:
        # Ask user if they want to record extra presence after reaching goal
        return self._ask_yes_no(
            "Meta Atingida",
            f"Você já registrou {self.manager.monthly_goal} vezes este mês.\nDeseja registrar mais uma por conta própria?"
        )

    def _ask_yes_no(self, title: str, question: str) -> bool:
        # General purpose Yes/No dialog
        dialog = BaseDialog()
        result = [False]

        def on_yes():
            result[0] = True
            dialog.destroy()

        def on_no():
            result[0] = False
            dialog.destroy()

        dialog.create_window(title)
        dialog.create_label(question).pack()

        frame = Frame(dialog.window)
        frame.pack(pady=10)
        dialog.create_button("Sim", on_yes).pack(in_=frame, side="left", padx=5)
        dialog.create_button("Não", on_no).pack(in_=frame, side="right", padx=5)

        dialog.window.grab_set()
        dialog.window.wait_window()
        return result[0]

    def _ask_area_selection_and_save(self, observation: str) -> None:
        # Area selection dialog with save operation
        from tkinter import StringVar

        dialog = BaseDialog()
        var = StringVar()

        def on_select():
            selected = var.get()
            if selected:
                dialog.destroy()
                self.manager.save_presence(True, observation, selected)
                MessageDialog().show("Registrado", "Sua resposta foi salva com sucesso.")
            else:
                error_label.config(text="Selecione uma área.", fg="red")

        dialog.create_window("Área")
        dialog.create_label("Em qual área você está hoje?").pack()

        for area in Area:
            Radiobutton(dialog.window, text=area.value, variable=var,
                        value=area.value).pack(anchor="w", padx=20)

        error_label = Label(dialog.window, text="", font=("Courier", 12))
        error_label.pack()

        Button(dialog.window, text="Confirmar", command=on_select,
               width=15, font=("Courier", 12)).pack(pady=10)

        dialog.window.grab_set()
        dialog.window.wait_window()

    def ask_presence(self) -> None:
        # Overall flow for presence interaction
        total, records = self.manager.count_monthly_presence()
        self._show_summary(total, records)

        if self._ask_is_present():
            self._handle_present_response(total)
        else:
            self._handle_absent_response()

    def _handle_present_response(self, total: int) -> None:
        # Handles logic for user marked as present
        if total >= self.manager.monthly_goal:
            if not self._ask_extra_confirmation():
                self.message_dialog.show("Encerrado", "Nenhum registro foi adicionado.")
                return
            observation = self.manager.config.EXTRA_LABEL
        else:
            observation = ""
        self._ask_area_selection_and_save(observation)

    def _handle_absent_response(self) -> None:
        # Handles logic for user marked as absent
        self.manager.save_presence(False)
        self.message_dialog.show("Informativo", "Tudo bem. Hoje não será contado como presencial.")


# Application entry point
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
