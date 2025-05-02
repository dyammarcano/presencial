import csv
import os
import platform
from datetime import datetime
from pathlib import Path
from typing import Optional
import tkinter as tk
from tkinter import messagebox


class PresenceTracker:
    # Configuration constants
    FOLDER_NAME = "Precencial"
    CSV_FILENAME = "registros.csv"
    CSV_HEADERS = ["data", "hora", "resposta", "observacao"]
    DEFAULT_MONTHLY_GOAL = 8

    # Response constants
    YES = "Sim"
    NO = "Não"
    EXTRA = "extra"

    def __init__(self, monthly_goal: int = DEFAULT_MONTHLY_GOAL):
        self.monthly_goal = monthly_goal
        self.data_folder = self._initialize_data_folder()
        self.csv_path = self.data_folder / self.CSV_FILENAME
        self._ensure_csv_exists()

    def _initialize_data_folder(self) -> Path:
        """Initialize and return the data folder path."""
        base_path = Path(os.environ["USERPROFILE"] if platform.system() == "Windows"
                         else os.path.expanduser("~"))
        folder_path = base_path / self.FOLDER_NAME
        folder_path.mkdir(exist_ok=True)
        return folder_path

    def _ensure_csv_exists(self) -> None:
        """Create a CSV file with headers if it doesn't exist."""
        if not self.csv_path.exists():
            with open(self.csv_path, mode='w', newline='') as file:
                writer = csv.writer(file)
                writer.writerow(self.CSV_HEADERS)

    def count_monthly_presence(self) -> int:
        """Count presence entries for the current month."""
        current = datetime.now()
        total = 0

        try:
            with open(self.csv_path, mode='r') as file:
                reader = csv.DictReader(file)
                for row in reader:
                    date = datetime.strptime(row["data"], "%Y-%m-%d")
                    if (date.year == current.year and
                            date.month == current.month and
                            row["resposta"] == self.YES):
                        total += 1
            return total
        except (IOError, csv.Error) as e:
            messagebox.showerror("Error", f"Failed to read presence data: {str(e)}")
            return 0

    def save_presence(self, is_present: bool, observation: str = "") -> None:
        """Save presence record to CSV file."""
        current = datetime.now()
        response = self.YES if is_present else self.NO

        try:
            with open(self.csv_path, mode='a', newline='') as file:
                writer = csv.writer(file)
                writer.writerow([
                    current.date(),
                    current.strftime("%H:%M:%S"),
                    response,
                    observation
                ])
            messagebox.showinfo("Success", "Your response has been recorded.")
        except IOError as e:
            messagebox.showerror("Error", f"Failed to save presence: {str(e)}")

    def prompt_presence(self) -> None:
        """Show presence prompt dialog and handle response."""
        monthly_count = self.count_monthly_presence()

        if monthly_count >= self.monthly_goal:
            if messagebox.askyesno(
                    "Goal Reached",
                    f"You've already recorded {self.monthly_goal} times this month.\n"
                    "Would you like to record an extra presence?"
            ):
                self.save_presence(True, self.EXTRA)
            else:
                messagebox.showinfo("Done", "No record was added.")
            return

        is_present = messagebox.askyesno("Presence", "Are you present today?")
        self.save_presence(is_present)


if __name__ == "__main__":
    root = tk.Tk()
    root.withdraw()
    tracker = PresenceTracker()
    tracker.prompt_presence()
