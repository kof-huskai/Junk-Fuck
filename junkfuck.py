#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Junk File Cleaner - Deep C: Drive Scanner with CLI Interface
Features:
- Centered colored output with spinner animation
- DPI awareness for all displays
- Discord and app protection
- Deep scanning with logging
- Table-formatted logs
- User confirmation for each deletion
"""

__version__ = "3.0.0"

import os
import sys
import platform
import shutil
import stat
import time
import threading
import ctypes
from pathlib import Path
from typing import List, Tuple, Set, Dict, Optional
from dataclasses import dataclass
from enum import Enum
import colorama
from colorama import Fore, Back, Style

colorama.init(autoreset=True)

if platform.system() == "Windows":
    try:
        ctypes.windll.shcore.SetProcessDpiAwareness(2)
    except:
        try:
            ctypes.windll.user32.SetProcessDPIAware()
        except:
            pass


class Colors:
    HEADER = Fore.CYAN + Style.BRIGHT
    SUCCESS = Fore.GREEN + Style.BRIGHT
    WARNING = Fore.YELLOW + Style.BRIGHT
    ERROR = Fore.RED + Style.BRIGHT
    INFO = Fore.BLUE + Style.BRIGHT
    MAGENTA = Fore.MAGENTA + Style.BRIGHT
    WHITE = Fore.WHITE + Style.BRIGHT
    DIM = Fore.WHITE + Style.DIM
    RESET = Style.RESET_ALL
    BOLD = Style.BRIGHT
    UNDERLINE = Style.DIM


class Spinner:
    def __init__(self, message: str = "Processing"):
        self.message = message
        self.spinner_chars = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"]
        self.running = False
        self.thread = None
        self.idx = 0

    def start(self):
        self.running = True
        self.thread = threading.Thread(target=self._spin)
        self.thread.daemon = True
        self.thread.start()

    def stop(self):
        self.running = False
        if self.thread:
            self.thread.join(timeout=0.5)
        self._clear_line()

    def _spin(self):
        while self.running:
            char = self.spinner_chars[self.idx % len(self.spinner_chars)]
            self.idx += 1
            cols = self._get_console_width()
            centered_msg = self._center_text(f"{Colors.INFO}{char} {self.message}{Colors.RESET}", cols)
            sys.stdout.write(f"\r{centered_msg}")
            sys.stdout.flush()
            time.sleep(0.08)

    def _clear_line(self):
        cols = self._get_console_width()
        sys.stdout.write(f"\r{' ' * cols}\r")
        sys.stdout.flush()

    def _get_console_width(self) -> int:
        try:
            return shutil.get_terminal_size().columns
        except:
            return 80

    def _center_text(self, text: str, width: int) -> str:
        visible_len = len(text.replace(Colors.INFO, "").replace(Colors.RESET, ""))
        if visible_len >= width:
            return text
        padding = (width - visible_len) // 2
        return " " * padding + text


@dataclass
class JunkItem:
    path: str
    name: str
    size: int
    is_folder: bool
    category: str

    def size_str(self) -> str:
        if self.size < 1024:
            return f"{self.size} B"
        elif self.size < 1024 * 1024:
            return f"{self.size / 1024:.2f} KB"
        elif self.size < 1024 * 1024 * 1024:
            return f"{self.size / (1024 * 1024):.2f} MB"
        else:
            return f"{self.size / (1024 * 1024 * 1024):.2f} GB"

    def short_name(self, max_len: int = 40) -> str:
        if len(self.name) <= max_len:
            return self.name
        return self.name[:max_len - 3] + "..."


class ConsoleUI:
    def __init__(self):
        self.width = self._get_width()
        self.spinner = None

    def _get_width(self) -> int:
        try:
            return shutil.get_terminal_size().columns
        except:
            return 100

    def update_width(self):
        self.width = self._get_width()

    def center(self, text: str, color: str = "") -> str:
        visible = text.replace(color, "").replace(Colors.RESET, "")
        if len(visible) >= self.width:
            return color + text + Colors.RESET
        padding = (self.width - len(visible)) // 2
        return " " * padding + color + text + Colors.RESET

    def print_center(self, text: str, color: str = ""):
        print(self.center(text, color))

    def print_line(self, char: str = "=", color: str = Colors.HEADER):
        print(self.center(char * self.width, color))

    def print_box(self, lines: List[str], color: str = Colors.HEADER, title: str = ""):
        self.update_width()
        content_width = min(self.width - 4, max(len(l.replace(Colors.RESET, "").replace(Colors.HEADER, "").replace(Colors.SUCCESS, "").replace(Colors.WARNING, "").replace(Colors.ERROR, "").replace(Colors.INFO, "").replace(Colors.MAGENTA, "")) for l in lines) + 2)
        if title:
            content_width = max(content_width, len(title) + 2)
        
        top = "+" + "-" * content_width + "+"
        bottom = "+" + "-" * content_width + "+"
        
        self.print_center(top, color)
        if title:
            self.print_center(f"| {title.center(content_width - 2)} |", color)
            self.print_center(f"+" + "-" * content_width + "+", color)
        for line in lines:
            visible = line.replace(Colors.RESET, "").replace(Colors.HEADER, "").replace(Colors.SUCCESS, "").replace(Colors.WARNING, "").replace(Colors.ERROR, "").replace(Colors.INFO, "").replace(Colors.MAGENTA, "")
            padding = content_width - len(visible)
            self.print_center(f"| {line}{' ' * padding} |", color)
        self.print_center(bottom, color)

    def start_spinner(self, message: str):
        self.spinner = Spinner(message)
        self.spinner.start()

    def stop_spinner(self):
        if self.spinner:
            self.spinner.stop()
            self.spinner = None

    def print_table(self, headers: List[str], rows: List[List[str]], col_colors: List[str] = None):
        self.update_width()
        col_count = len(headers)
        col_widths = [len(h) for h in headers]
        
        for row in rows:
            for i, cell in enumerate(row):
                visible = cell.replace(Colors.RESET, "").replace(Colors.SUCCESS, "").replace(Colors.WARNING, "").replace(Colors.ERROR, "").replace(Colors.INFO, "").replace(Colors.MAGENTA, "")
                col_widths[i] = max(col_widths[i], len(visible))
        
        total_width = sum(col_widths) + (col_count - 1) * 3 + 4
        if total_width > self.width:
            scale = (self.width - 4) / total_width
            col_widths = [max(8, int(w * scale)) for w in col_widths]
        
        def fmt_row(cells, colors=None):
            parts = []
            for i, cell in enumerate(cells):
                visible = cell.replace(Colors.RESET, "").replace(Colors.SUCCESS, "").replace(Colors.WARNING, "").replace(Colors.ERROR, "").replace(Colors.INFO, "").replace(Colors.MAGENTA, "")
                padding = col_widths[i] - len(visible)
                c = colors[i] if colors else ""
                parts.append(f" {c}{cell}{Colors.RESET}{' ' * padding} ")
            return "|" + "|".join(parts) + "|"
        
        header_line = fmt_row(headers, [Colors.BOLD] * col_count)
        sep = "+" + "+".join("-" * (w + 2) for w in col_widths) + "+"
        
        self.print_center("+" + "+".join("-" * (w + 2) for w in col_widths) + "+", Colors.HEADER)
        self.print_center(header_line, Colors.HEADER)
        self.print_center(sep, Colors.HEADER)
        
        for row in rows:
            self.print_center(fmt_row(row, col_colors), Colors.RESET)
        
        self.print_center("+" + "+".join("-" * (w + 2) for w in col_widths) + "+", Colors.HEADER)

    def prompt_center(self, prompt: str, color: str = Colors.WARNING) -> str:
        self.update_width()
        centered = self.center(prompt, color)
        return input(centered).strip()

    def confirm_center(self, message: str) -> bool:
        while True:
            choice = self.prompt_center(f"{message} (y/n): ", Colors.WARNING).lower()
            if choice in ['y', 'yes']:
                return True
            elif choice in ['n', 'no']:
                return False
            self.print_center("Please enter 'y' or 'n'", Colors.ERROR)


class JunkCleaner:
    PROTECTED_APPS = {
        'discord', 'discord ptb', 'discord canary', 'discord development',
        'slack', 'teams', 'zoom', 'skype', 'telegram', 'whatsapp',
        'spotify', 'steam', 'epicgames', 'battle.net', 'origin', 'uplay',
        'chrome', 'firefox', 'edge', 'brave', 'opera', 'vivaldi',
        'vscode', 'code', 'pycharm', 'intellij', 'webstorm', 'rider',
        'postman', 'insomnia', 'docker', 'vmware', 'virtualbox',
        'obs', 'streamlabs', 'xsplit', 'nvidia', 'amd', 'intel',
        'logitech', 'razer', 'steelseries', 'corsair',
    }

    PROTECTED_PATHS = {
        'C:\\Windows',
        'C:\\Windows\\System32',
        'C:\\Windows\\SysWOW64',
        'C:\\Program Files',
        'C:\\Program Files (x86)',
        'C:\\ProgramData',
        'C:\\Users\\All Users',
        'C:\\System Volume Information',
        'C:\\$Recycle.Bin',
        'C:\\Recovery',
        'C:\\Boot',
        'C:\\Documents and Settings',
    }

    JUNK_EXTENSIONS = {
        '.tmp', '.temp', '.log', '.old', '.bak', '.backup',
        '.dmp', '.dump', '.cache', '.thumb', '.thumbcache',
        '.chk', '.fts', '.gid', '.pft', '.~', '.~~~',
        '.msi', '.cab', '.iso', '.nrg', '.dmg',
        '.part', '.crdownload', '.download',
        '.pyc', '.pyo', '.class', '.o', '.obj',
        '.exe~', '.dll~', '.so~', '.dylib~',
        '.swp', '.swo', '.swm', '.~tmp',
        '.tmp_', '_temp', '_backup',
        '.sess', '.session', '.lock',
        '.error', '.trace', '.stackdump',
        '.etl', '.evtx', '.wer', '.hdmp', '.mdmp',
    }

    JUNK_FOLDERS = {
        '__pycache__', '.cache', '.tmp', 'temp', 'Temp', 'tmp',
        'Cache', 'caches', 'Logs', 'logs', 'Backup', 'backup',
        '.trash', 'Trash', 'Prefetch', 'Temporary', 'Thumbnails',
        'thumbnails', 'cached', 'Cached', '.thumbnails', '.Trash-1000',
        'Temp', 'TMP', 'Downloaded Installations', 'DeliveryOptimization',
    }

    def __init__(self, ui: ConsoleUI):
        self.ui = ui
        self.system = platform.system()
        self.deleted_count = 0
        self.skipped_count = 0
        self.failed_count = 0
        self.total_freed = 0
        self.protected_paths = self._build_protected_paths()
        self.junk_items: List[JunkItem] = []

    def _build_protected_paths(self) -> Set[str]:
        protected = set(self.PROTECTED_PATHS)
        
        if self.system == "Windows":
            try:
                system_root = os.environ.get('SYSTEMROOT', '')
                windir = os.environ.get('WINDIR', '')
                program_files = os.environ.get('ProgramFiles', '')
                program_files_x86 = os.environ.get('ProgramFiles(x86)', '')
                program_data = os.environ.get('ProgramData', '')
                appdata = os.environ.get('APPDATA', '')
                localappdata = os.environ.get('LOCALAPPDATA', '')
                userprofile = os.environ.get('USERPROFILE', '')
                
                for p in [system_root, windir, program_files, program_files_x86, program_data, appdata, localappdata, userprofile]:
                    if p:
                        protected.add(p)
                
                if userprofile:
                    for app in self.PROTECTED_APPS:
                        app_paths = [
                            os.path.join(localappdata, app),
                            os.path.join(appdata, app),
                            os.path.join(userprofile, 'AppData', 'Local', app),
                            os.path.join(userprofile, 'AppData', 'Roaming', app),
                        ]
                        for ap in app_paths:
                            protected.add(ap)
            except:
                pass
        
        return {os.path.abspath(p).lower() for p in protected if p}

    def _is_protected(self, path: str) -> bool:
        try:
            abs_path = os.path.abspath(path).lower()
            
            if abs_path in ['c:\\', 'd:\\', 'e:\\', 'f:\\', '/']:
                return True
            
            for protected in self.protected_paths:
                if abs_path == protected or abs_path.startswith(protected + os.sep):
                    return True
            
            home = os.path.abspath(os.path.expanduser("~")).lower()
            if abs_path == home:
                return True
            
            name = os.path.basename(abs_path).lower()
            for app in self.PROTECTED_APPS:
                if app in name:
                    return True
            
            return False
        except:
            return True

    def _get_size(self, path: str) -> int:
        try:
            if os.path.isfile(path):
                return os.path.getsize(path)
            elif os.path.isdir(path):
                total = 0
                for dirpath, dirnames, filenames in os.walk(path):
                    for f in filenames:
                        fp = os.path.join(dirpath, f)
                        try:
                            if os.path.isfile(fp):
                                total += os.path.getsize(fp)
                        except:
                            pass
                return total
        except:
            pass
        return 0

    def _is_junk_file(self, file_path: str) -> bool:
        try:
            if not os.path.isfile(file_path):
                return False
            if self._is_protected(file_path):
                return False
            
            name = os.path.basename(file_path)
            ext = os.path.splitext(name)[1].lower()
            
            if ext in self.JUNK_EXTENSIONS:
                return True
            
            name_lower = name.lower()
            junk_keywords = ['temp', 'tmp', 'cache', 'backup', 'log', 'old', '~', 'copy', 'crash', 'dump', 'error']
            for kw in junk_keywords:
                if kw in name_lower:
                    return True
            
            return False
        except:
            return False

    def _is_junk_folder(self, folder_path: str) -> bool:
        try:
            if not os.path.isdir(folder_path):
                return False
            if self._is_protected(folder_path):
                return False
            
            name = os.path.basename(folder_path)
            if name in self.JUNK_FOLDERS:
                return True
            
            name_lower = name.lower()
            if name_lower in ['temp', 'tmp', 'cache', 'logs', 'trash', 'backup', 'thumbnail']:
                return True
            
            return False
        except:
            return False

    def _get_category(self, path: str, is_folder: bool) -> str:
        name = os.path.basename(path).lower()
        ext = os.path.splitext(name)[1].lower()
        
        if is_folder:
            if 'cache' in name:
                return "Cache"
            elif 'temp' in name or 'tmp' in name:
                return "Temp"
            elif 'log' in name:
                return "Logs"
            elif 'backup' in name:
                return "Backup"
            elif 'trash' in name:
                return "Trash"
            elif 'thumb' in name:
                return "Thumbnails"
            elif 'prefetch' in name:
                return "Prefetch"
            return "Junk Folder"
        
        if ext in ['.tmp', '.temp', '.cache']:
            return "Temp"
        elif ext in ['.log', '.etl', '.evtx']:
            return "Logs"
        elif ext in ['.bak', '.backup', '.old']:
            return "Backup"
        elif ext in ['.dmp', '.dump', '.hdmp', '.mdmp', '.wer']:
            return "Crash Dumps"
        elif ext in ['.crdownload', '.part', '.download']:
            return "Partial Downloads"
        elif ext in ['.pyc', '.pyo', '.class', '.o', '.obj']:
            return "Build Artifacts"
        elif ext in ['.swp', '.swo', '.~tmp']:
            return "Editor Temp"
        return "Junk File"

    def scan_drive(self, drive: str = "C:\\") -> List[JunkItem]:
        self.ui.print_center(f"[SCAN] Scanning {drive} for junk files...")
        junk_items = []
        
        try:
            for root, dirs, files in os.walk(drive):
                if self._is_protected(root):
                    dirs[:] = []
                    continue
                
                for dir_name in dirs[:]:
                    dir_path = os.path.join(root, dir_name)
                    if self._is_junk_folder(dir_path):
                        size = self._get_size(dir_path)
                        junk_items.append(JunkItem(
                            path=dir_path,
                            name=dir_name,
                            size=size,
                            is_folder=True,
                            category=self._get_category(dir_path, True)
                        ))
                        dirs.remove(dir_name)
                
                for file_name in files:
                    file_path = os.path.join(root, file_name)
                    if self._is_junk_file(file_path):
                        size = self._get_size(file_path)
                        junk_items.append(JunkItem(
                            path=file_path,
                            name=file_name,
                            size=size,
                            is_folder=False,
                            category=self._get_category(file_path, False)
                        ))
        except (PermissionError, OSError):
            pass
        
        self.ui.stop_spinner()
        self.junk_items = sorted(junk_items, key=lambda x: (x.is_folder, -x.size))
        return self.junk_items

    def print_log_table(self):
        if not self.junk_items:
            self.ui.print_center("No junk files found!", Colors.SUCCESS)
            return
        
        self.ui.print_center(f"\nFound {len(self.junk_items)} junk items", Colors.INFO)
        
        headers = ["#", "Name", "Size", "Type", "Category"]
        rows = []
        for idx, item in enumerate(self.junk_items, 1):
            rows.append([
                str(idx),
                item.short_name(45),
                item.size_str(),
                "[Folder]" if item.is_folder else "[File]",
                item.category
            ])
        
        self.ui.print_table(headers, rows, [Colors.DIM, Colors.WHITE, Colors.SUCCESS, Colors.INFO, Colors.MAGENTA])

    def confirm_and_delete(self, item: JunkItem) -> bool:
        self.ui.update_width()
        self.ui.print_line("─", Colors.WARNING)
        self.ui.print_center(f"Item: {item.name}", Colors.WHITE)
        self.ui.print_center(f"Path: {item.path}", Colors.DIM)
        self.ui.print_center(f"Size: {item.size_str()}  |  Type: {'Folder' if item.is_folder else 'File'}  |  Category: {item.category}", Colors.INFO)
        self.ui.print_line("─", Colors.WARNING)
        
        if self.ui.confirm_center("Delete this item?"):
            return self.delete_item(item)
        else:
            self.skipped_count += 1
            self.ui.print_center(f"⏭ Skipped: {item.name}", Colors.WARNING)
            return False

    def delete_item(self, item: JunkItem) -> bool:
        try:
            if self._is_protected(item.path):
                self.ui.print_center(f"⚠ Protected: {item.name} - Skipped", Colors.ERROR)
                self.skipped_count += 1
                return False
            
            if item.is_folder:
                shutil.rmtree(item.path, onerror=self._remove_readonly)
                self.ui.print_center(f"✅ Deleted folder: {item.name} ({item.size_str()})", Colors.SUCCESS)
            else:
                os.remove(item.path)
                self.ui.print_center(f"✅ Deleted file: {item.name} ({item.size_str()})", Colors.SUCCESS)
            
            self.deleted_count += 1
            self.total_freed += item.size
            return True
        except Exception as e:
            self.failed_count += 1
            self.ui.print_center(f"❌ Failed to delete {item.name}: {e}", Colors.ERROR)
            return False

    def _remove_readonly(self, func, path, exc_info):
        os.chmod(path, stat.S_IWRITE)
        func(path)

    def run_interactive(self):
        #self.ui.print_line("=", Colors.HEADER)
        self.ui.print_center("     ██╗██╗   ██╗███╗   ██╗██╗  ██╗    ███████╗██╗   ██╗ ██████╗██╗  ██╗ ", Colors.HEADER)
        self.ui.print_center("     ██║██║   ██║████╗  ██║██║ ██╔╝    ██╔════╝██║   ██║██╔════╝██║ ██╔╝ ", Colors.HEADER)
        self.ui.print_center("     ██║██║   ██║██╔██╗ ██║█████╔╝     █████╗  ██║   ██║██║     █████╔╝  ", Colors.HEADER)
        self.ui.print_center("██   ██║██║   ██║██║╚██╗██║██╔═██╗     ██╔══╝  ██║   ██║██║     ██╔═██╗  ", Colors.HEADER)
        self.ui.print_center("╚█████╔╝╚██████╔╝██║ ╚████║██║  ██╗    ██║     ╚██████╔╝╚██████╗██║  ██╗ ", Colors.HEADER)
        self.ui.print_center(" ╚════╝  ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝    ╚═╝      ╚═════╝  ╚═════╝╚═╝  ╚═╝ ", Colors.HEADER)
        self.ui.print_center("                                JUNKFUCK v3.0                            ", Colors.MAGENTA)
        self.ui.print_center("                      Deep C: Drive Scanner & Cleaner                    ", Colors.INFO)
        self.ui.print_center("                      -------------------------------                    ", Colors.HEADER)
        
        self.ui.print_center(f"System: {self.system}  |  Drive: C:\\", Colors.INFO)
        self.ui.print_center("Protected: Discord, Browsers, IDEs, Games, System Paths", Colors.WARNING)
        self.ui.print_line("─", Colors.DIM)
        
        if self.system == "Windows":
            try:
                is_admin = ctypes.windll.shell32.IsUserAnAdmin()
                if not is_admin:
                    self.ui.print_center("⚠ Not running as Administrator - some files may be inaccessible", Colors.WARNING)
            except:
                pass
        
        self.ui.print_center("")
        
        items = self.scan_drive("C:\\")
        
        if not items:
            self.ui.print_center("✅ No junk files found on C: drive!", Colors.SUCCESS)
            return
        
        self.print_log_table()
        
        self.ui.print_line("─", Colors.WARNING)
        total_size = sum(i.size for i in items)
        self.ui.print_center(f"Total items: {len(items)}  |  Total size: {self._format_size(total_size)}", Colors.INFO)
        self.ui.print_center("You will be asked to confirm each item for deletion.", Colors.WARNING)
        self.ui.print_line("─", Colors.WARNING)
        
        if not self.ui.confirm_center("Proceed with interactive deletion?"):
            self.ui.print_center("Operation cancelled.", Colors.INFO)
            return
        
        for item in items:
            self.confirm_and_delete(item)
        
        self.print_final_report()

    def print_final_report(self):
        self.ui.print_line("=", Colors.HEADER)
        self.ui.print_center("+===============================================================+", Colors.HEADER)
        self.ui.print_center("|                        FINAL REPORT                           |", Colors.HEADER)
        self.ui.print_center("+===============================================================+", Colors.HEADER)
        
        report = [
            f"[DELETED]  Deleted:   {self.deleted_count} items",
            f"[SKIPPED]  Skipped:   {self.skipped_count} items",
            f"[FAILED]   Failed:    {self.failed_count} items",
            f"[FREED]    Space freed: {self._format_size(self.total_freed)}",
        ]
        for line in report:
            self.ui.print_center(line, Colors.SUCCESS if "Deleted" in line or "freed" in line else Colors.WARNING if "Skipped" in line else Colors.ERROR)
        
        self.ui.print_line("═", Colors.HEADER)
        self.ui.print_center("Operation completed. Press Enter to exit...", Colors.INFO)
        input()

    def _format_size(self, bytes_size: int) -> str:
        if bytes_size < 1024:
            return f"{bytes_size} B"
        elif bytes_size < 1024 * 1024:
            return f"{bytes_size / 1024:.2f} KB"
        elif bytes_size < 1024 * 1024 * 1024:
            return f"{bytes_size / (1024 * 1024):.2f} MB"
        else:
            return f"{bytes_size / (1024 * 1024 * 1024):.2f} GB"


def main():
    try:
        ui = ConsoleUI()
        cleaner = JunkCleaner(ui)
        cleaner.run_interactive()
    except KeyboardInterrupt:
        ui = ConsoleUI()
        ui.print_center("\n⏹ Operation interrupted by user.", Colors.WARNING)
        input("\nPress Enter to exit...")
    except Exception as e:
        ui = ConsoleUI()
        ui.print_center(f"\n❌ Unexpected error: {e}", Colors.ERROR)
        import traceback
        traceback.print_exc()
        input("\nPress Enter to exit...")


if __name__ == "__main__":
    main()
    ui = ConsoleUI()
    ui.print_center("\nPress Enter to exit...", Colors.INFO)
    input()