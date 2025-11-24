# Specter - RSI Store Checkout Assistant

[English](#english) | [Русский](#russian)

---

<a name="english"></a>
# English Documentation

## Table of Contents
- [Overview](#overview)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [First-Time Setup](#first-time-setup)
- [Configuration](#configuration)
- [Command-Line Options](#command-line-options)
- [Usage Examples](#usage-examples)
- [How It Works](#how-it-works)
- [Tips for Maximum Speed](#tips-for-maximum-speed)
- [Troubleshooting](#troubleshooting)
- [Legal Disclaimer](#legal-disclaimer)

## Overview

Specter is a high-performance automation tool designed for purchasing limited-availability ships and items on RobertsSpaceIndustries.com. Built with Go and the Rod browser automation library, it provides speed and reliability when milliseconds matter.

**Speed is critical**: Specter is optimized for wave-based limited ship sales where items sell out in seconds. Every delay has been minimized while maintaining reliability.

## Features

- ⚡ **Blazing Fast**: Optimized timing with minimal waits (100-500ms between actions)
- 🖥️ **Cross-Platform**: Runs on macOS (Intel & Apple Silicon) and Windows
- 🔄 **Auto-Retry**: Configurable retry logic with 30+ attempts by default
- 💳 **Smart Credit Application**: Automatically applies store credit
- 🎯 **Skip-Cart Mode**: Jump directly to checkout if item already in cart
- 🧪 **Dry-Run Mode**: Test the flow without completing purchase
- 🔍 **Debug Mode**: Detailed logging for troubleshooting
- 👤 **Session Persistence**: Maintains login between runs
- 🎲 **Randomized Delays**: Human-like behavior to avoid detection

## Prerequisites

### To Run the Application
- Chrome, Edge, or Firefox browser installed
- Active RSI account (you must login before the sale)
- Fast internet connection (wired recommended)

### To Build from Source
- Go 1.21 or later
- Git (optional)

## Installation

### Option 1: Download Pre-built Binary (Recommended)

1. Download the latest release for your platform from the [Releases]() page
2. Extract the archive to a folder
3. Open terminal/command prompt in that folder

### Option 2: Build from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/specter.git
   cd specter
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build:
   ```bash
   # macOS/Linux
   go build -o specter

   # Windows
   go build -o specter.exe
   ```

## First-Time Setup

**IMPORTANT**: Complete this setup BEFORE the ship sale begins!

### Step 1: Initial Login

1. Run the app for the first time:
   ```bash
   ./specter
   ```

2. A browser window will open. The app will show an error ("No item URL specified") - **this is expected**.

3. **In the browser window that opened**: Log in to your RSI account

4. **Leave the browser window open** and let the app exit with the error

5. Your login session is now saved!

### Step 2: Configure Target Item

1. Open `config.yaml` in a text editor

2. Set the URL of the ship/item you want to purchase:
   ```yaml
   item_url: "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"
   ```

3. **(Optional)** Adjust timing settings for maximum speed (see [Tips for Maximum Speed](#tips-for-maximum-speed))

### Step 3: Test Run

**Do a test run with a regular (non-limited) ship before the actual sale!**

```bash
# Test with dry-run mode (won't complete purchase)
./specter -dry-run

# Or test with a cheap item you don't mind buying
./specter -url "https://robertsspaceindustries.com/pledge/Standalone-Ships/Aurora-ES"
```

This verifies everything works and warms up your browser profile.

## Configuration

The `config.yaml` file controls all settings. A default configuration is created on first run.

### Complete Configuration Example

```yaml
# Target item URL
item_url: "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"

# Browser settings
browser_profile_path: /Users/username/.specter/browser-profile
browser_type: chrome                    # chrome, edge, or firefox
headless: false                         # true = no visible window
keep_browser_open: true                 # Keep browser open after completion

# Timing settings (in seconds)
page_load_timeout: 30                   # Max time to wait for page loads
min_delay_between: 0.5                  # Minimum delay between actions
max_delay_between: 1.0                  # Maximum delay between actions
checkout_ready_delay: 2                 # Delay before starting checkout (now ignored in code)

# Retry settings
max_retries: 30                         # Number of retry attempts
retry_delay_min: 0.4                    # Min delay between retries
retry_delay_max: 0.8                    # Max delay between retries

# Feature flags
auto_apply_credit: true                 # Automatically apply store credit
dry_run: false                          # Test mode (stops before final purchase)
interactive: false                      # Pause at each step for review
debug_mode: false                       # Enable detailed logging

# Display settings
viewport_width: 1920
viewport_height: 1080

# CSS Selectors (advanced - only change if site structure changes)
selectors:
    add_to_cart_button: .m-storeAction__button
```

### Configuration Options Reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `item_url` | string | "" | Direct URL to the item to purchase |
| `browser_profile_path` | string | ~/.specter/browser-profile | Browser session storage location |
| `browser_type` | string | "chrome" | Browser: chrome, edge, or firefox |
| `headless` | bool | false | Run without visible window (slightly faster) |
| `keep_browser_open` | bool | true | Keep browser open after completion |
| `page_load_timeout` | int | 30 | Max seconds to wait for pages |
| `min_delay_between` | float | 0.5 | Min seconds between actions |
| `max_delay_between` | float | 1.0 | Max seconds between actions |
| `checkout_ready_delay` | int | 2 | Initial delay (deprecated, now ignored) |
| `max_retries` | int | 30 | Retry attempts if checkout fails |
| `retry_delay_min` | float | 0.4 | Min seconds between retries |
| `retry_delay_max` | float | 0.8 | Max seconds between retries |
| `auto_apply_credit` | bool | true | Auto-apply store credit |
| `dry_run` | bool | false | Test mode (stops before purchase) |
| `interactive` | bool | false | Pause at each step |
| `debug_mode` | bool | false | Detailed logging |
| `viewport_width` | int | 1920 | Browser window width |
| `viewport_height` | int | 1080 | Browser window height |

## Command-Line Options

Command-line flags override config file settings.

### Available Flags

```bash
-config <path>        # Path to config file (default: config.yaml)
-url <url>            # Target item URL (overrides config)
-dry-run              # Test mode: stops before clicking "I agree"
-interactive          # Pauses at each step for manual review
-debug                # Enables detailed debug logging
-skip-cart            # Skip add-to-cart step (item already in cart)
```

### Flag Details

#### `-config <path>`
Use a custom configuration file. Useful for managing multiple target items.

**Example:**
```bash
./specter -config idris-config.yaml
```

#### `-url <url>`
Specify target URL directly without editing config file.

**Example:**
```bash
./specter -url "https://robertsspaceindustries.com/pledge/ships/anvil-carrack/Carrack"
```

#### `-dry-run`
Test the entire flow without completing the purchase. The app will:
- Navigate to the item
- Add to cart
- Proceed through checkout
- Apply store credit
- Accept terms
- **Stop before clicking "I agree"** (the final purchase button)

**Example:**
```bash
./specter -dry-run
```

**Use this to**: Verify everything works before the actual sale.

#### `-interactive`
Pauses at each step and waits for you to press Enter. Useful for:
- Understanding the flow
- Verifying each step manually
- Debugging issues

**Example:**
```bash
./specter -interactive
```

#### `-debug`
Enables detailed logging showing:
- JavaScript evaluation results
- Timing information
- Element search attempts
- Internal state changes

**Example:**
```bash
./specter -debug
```

**Use this to**: Troubleshoot issues or understand what's happening.

#### `-skip-cart`
Skips the "add to cart" step and goes directly to checkout. Use when:
- Item is already in your cart from a previous attempt
- You manually added the item and want to use the app for checkout only
- You're retrying after a failed attempt

**Example:**
```bash
./specter -skip-cart
```

**Note**: The app will navigate directly to the cart/checkout page.

### Combining Flags

You can combine multiple flags:

```bash
# Test run with debug logging
./specter -dry-run -debug

# Skip to checkout with custom URL
./specter -skip-cart -url "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"

# Interactive mode with debug logging
./specter -interactive -debug

# Production run with specific config
./specter -config production-config.yaml -url "https://robertsspaceindustries.com/pledge/ships/anvil-carrack/Carrack"
```

## Usage Examples

### Basic Usage Scenarios

#### 1. First-Time Test (Before the Sale)
```bash
# Complete dry-run test
./specter -dry-run -debug
```
This verifies your setup without completing a purchase.

#### 2. Production Run (During the Sale)
```bash
# Use config file settings
./specter

# Or specify URL directly
./specter -url "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"
```

#### 3. Retry After Failed Attempt (Item Still in Cart)
```bash
# Skip add-to-cart and go straight to checkout
./specter -skip-cart
```

#### 4. Multiple Target Items
Create separate config files:

**idris.yaml:**
```yaml
item_url: "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"
# ... other settings
```

**javelin.yaml:**
```yaml
item_url: "https://robertsspaceindustries.com/pledge/ships/aegis-javelin/Javelin"
# ... other settings
```

Then run:
```bash
./specter -config idris.yaml
# or
./specter -config javelin.yaml
```

#### 5. Troubleshooting Run
```bash
# See exactly what's happening
./specter -debug -interactive
```

### Pre-Sale Checklist

Complete this checklist 15-30 minutes before the sale:

```bash
# 1. Verify login is still active
./specter -dry-run

# 2. If login expired, browser will open - log in again

# 3. Do a full test run
./specter -dry-run -debug

# 4. Verify config is correct
cat config.yaml | grep item_url
cat config.yaml | grep dry_run  # Should be: false

# 5. Close all other applications to free resources

# 6. Ready for production!
./specter
```

## How It Works

### Complete Flow

1. **Browser Launch** (1-2 seconds)
   - Launches browser with saved profile
   - Maintains your login session

2. **Navigation** (100-500ms)
   - Navigates to target item URL
   - Extracts item price for verification

3. **Add to Cart** (100-300ms)
   - Finds and clicks "Add to cart" button
   - Waits for cart to update

4. **Navigate to Cart** (300-500ms)
   - Goes to shopping cart page
   - Verifies item is present

5. **Proceed to Checkout** (400-600ms)
   - Clicks checkout button
   - Waits for checkout page

6. **Apply Store Credit** (600-900ms if applicable)
   - Checks if total is already $0
   - If not, clicks credit input and types amount
   - Waits for credit to apply

7. **Proceed to Payment** (variable)
   - Checks if already at step 2
   - Clicks "Proceed to pay" if needed
   - Waits for disclaimer modal

8. **Accept Terms** (400-600ms)
   - Clicks "Jump to bottom" in modal
   - Checks agreement checkbox
   - Waits for button to enable

9. **Finalize** (instant)
   - Clicks "I agree" button
   - Purchase complete!

**Total typical time**: 3-6 seconds from start to completion (after browser launch)

### Retry Logic

If checkout fails (item unavailable, timeout, etc.):
- Automatically retries up to 30 times (configurable)
- Waits 400-800ms between attempts
- On retry, goes back to step 1 and tries again
- Tracks if item is already in cart to skip step 3 on retries

## Tips for Maximum Speed

### Critical Settings for Speed

Edit your `config.yaml`:

```yaml
# Absolute minimum delays (use at your own risk)
min_delay_between: 0.3
max_delay_between: 0.5
retry_delay_min: 0.2
retry_delay_max: 0.4

# Disable unnecessary features
keep_browser_open: false
debug_mode: false

# Optional: slightly faster (no visible window)
headless: true
```

### Speed Optimization Checklist

**Before the sale:**
- ✅ Use wired ethernet connection (not WiFi)
- ✅ Close all other applications
- ✅ Close all other browser windows/tabs
- ✅ Disable antivirus temporarily (if comfortable)
- ✅ Use headless mode: `headless: true`
- ✅ Reduce all timing values to minimum
- ✅ Set `keep_browser_open: false`
- ✅ Increase max_retries: `max_retries: 50`

**Browser choice:**
- Chrome is fastest and most reliable
- Edge is comparable to Chrome
- Firefox is slightly slower

**System:**
- Close Discord, Slack, Steam, etc.
- Disable system notifications
- Free up RAM (close unnecessary apps)

### Balanced vs Aggressive Settings

**Balanced (Recommended):**
```yaml
min_delay_between: 0.5
max_delay_between: 1.0
max_retries: 30
headless: false
```
- Reliable and reasonably fast
- Low risk of bot detection
- Good for most sales

**Aggressive (Maximum Speed):**
```yaml
min_delay_between: 0.2
max_delay_between: 0.4
max_retries: 50
headless: true
```
- Absolute fastest possible
- Higher risk of bot detection (use with caution)
- For highly contested sales only

## Troubleshooting

### Browser Issues

#### "Failed to get the debug url: Opening in existing browser session"
**Cause**: Another instance is using the browser profile.

**Solution**:
```bash
# Kill any running instances
pkill -f specter

# On Windows, use Task Manager to end specter.exe

# Then run again
./specter
```

#### "Browser launch failed"
**Cause**: Browser not found or profile corrupted.

**Solutions**:
1. Ensure Chrome/Edge/Firefox is installed
2. Try deleting browser profile:
   ```bash
   # macOS/Linux
   rm -rf ~/.specter/browser-profile

   # Windows
   rmdir /s %USERPROFILE%\.specter\browser-profile
   ```
3. Run again and log in

### Checkout Issues

#### "Could not find add to cart button"
**Cause**: Website HTML changed or item unavailable.

**Solutions**:
1. Check if item is actually for sale
2. Verify URL is correct
3. Try updating the selector in config:
   ```yaml
   selectors:
       add_to_cart_button: ".your-custom-selector"
   ```
4. Run with `-debug` to see what's happening

#### "Failed to apply store credit"
**Cause**: Credit input not found or total already $0.

**Solutions**:
1. Verify you have store credit in your account
2. Check if item price is $0 (credit may already be applied)
3. Set `auto_apply_credit: false` and apply manually

#### App stops at checkout page
**Cause**: Disclaimer modal not appearing or timing issue.

**Solutions**:
1. Run with `-interactive` to manually step through
2. Check if you're actually logged in
3. Increase page_load_timeout: `page_load_timeout: 60`
4. Use `-debug` to see detailed logs

### Speed Issues

#### App seems slow
**Causes and solutions**:
1. Check timing values in config - reduce them
2. Ensure fast internet connection
3. Use `headless: true` for slight speed boost
4. Close other applications
5. Check if browser extensions are enabled (they slow things down)

#### Retries happening immediately
**Expected behavior**: The app retries automatically when checkout fails.

**To see why it's retrying**:
```bash
./specter -debug
```
Look for error messages showing why checkout failed.

### Login Issues

#### App doesn't remember login
**Cause**: Browser profile not saved or corrupted.

**Solution**:
1. Check browser_profile_path in config
2. Ensure the directory exists and is writable
3. Log in again and let browser save the session
4. Verify `~/.specter/browser-profile` contains files

#### Session expired during sale
**Prevention**:
- Log in 15 minutes before the sale
- Do a test run to warm up the session
- Keep the browser profile active

### Debug Mode

For any unexplained issue, run with full debugging:

```bash
./specter -debug -interactive
```

This will:
- Show all internal operations
- Pause at each step
- Display JavaScript evaluation results
- Show timing information

Share the debug output when asking for help.

## Browser Profile Location

Your login session is stored in:

**macOS/Linux:**
```
~/.specter/browser-profile
```

**Windows:**
```
%USERPROFILE%\.specter\browser-profile
```

**To reset (logout)**:
```bash
# macOS/Linux
rm -rf ~/.specter/browser-profile

# Windows (Command Prompt)
rmdir /s %USERPROFILE%\.specter\browser-profile

# Windows (PowerShell)
Remove-Item -Recurse -Force $env:USERPROFILE\.specter\browser-profile
```

## Legal Disclaimer

⚠️ **IMPORTANT**: Please read carefully.

This tool is for **personal use only**. Using automation tools may violate the Terms of Service of robertsspaceindustries.com. Use at your own risk.

**By using this software, you acknowledge that**:
- You are solely responsible for your use of this tool
- Automated purchasing may provide an unfair advantage over other users
- You should review and comply with RSI's Terms of Service
- The authors are not responsible for any consequences, including account suspension or termination
- This software is provided "as is" without warranty of any kind

**No Data Collection**: This application does not collect, store, or transmit any personal data. Everything runs locally on your machine.

**Security**: Your RSI credentials are handled only by the browser itself, never by this application. The app only sends browser automation commands.

## Building from Source

### Dependencies

```bash
go mod download
```

### Build Commands

```bash
# Current platform
go build -o specter

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o specter-darwin-amd64

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o specter-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o specter.exe

# Linux
GOOS=linux GOARCH=amd64 go build -o specter-linux
```

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## Support

- 🐛 **Issues**: Open an issue on GitHub
- 💬 **Questions**: Check existing issues first
- 📖 **Documentation**: This README

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Rod](https://github.com/go-rod/rod) - High-performance browser automation
- Inspired by the need for fair access to limited-time items

---

<a name="russian"></a>
# Русская документация

## Содержание
- [Обзор](#обзор)
- [Возможности](#возможности)
- [Требования](#требования)
- [Установка](#установка)
- [Первоначальная настройка](#первоначальная-настройка)
- [Конфигурация](#конфигурация)
- [Параметры командной строки](#параметры-командной-строки)
- [Примеры использования](#примеры-использования)
- [Как это работает](#как-это-работает)
- [Советы для максимальной скорости](#советы-для-максимальной-скорости)
- [Устранение неполадок](#устранение-неполадок)
- [Правовая оговорка](#правовая-оговорка)

## Обзор

Specter - это высокопроизводительный инструмент автоматизации, предназначенный для покупки кораблей и предметов ограниченной доступности на RobertsSpaceIndustries.com. Создан на Go с использованием библиотеки автоматизации браузера Rod, обеспечивает скорость и надежность, когда на счету каждая миллисекунда.

**Скорость критична**: Specter оптимизирован для волновых распродаж кораблей ограниченного выпуска, где товары распродаются за секунды. Каждая задержка минимизирована при сохранении надежности.

## Возможности

- ⚡ **Молниеносная скорость**: Оптимизированные задержки 100-500мс между действиями
- 🖥️ **Кроссплатформенность**: Работает на macOS (Intel и Apple Silicon) и Windows
- 🔄 **Автоповтор**: Настраиваемая логика повторов (по умолчанию 30+ попыток)
- 💳 **Умное применение кредитов**: Автоматически применяет store credit
- 🎯 **Режим Skip-Cart**: Прямой переход к оформлению, если товар уже в корзине
- 🧪 **Тестовый режим**: Проверка процесса без завершения покупки
- 🔍 **Режим отладки**: Подробное логирование для диагностики
- 👤 **Сохранение сессии**: Поддерживает вход между запусками
- 🎲 **Случайные задержки**: Человекоподобное поведение для избежания обнаружения

## Требования

### Для запуска приложения
- Установленный браузер Chrome, Edge или Firefox
- Активный аккаунт RSI (необходимо войти до распродажи)
- Быстрое интернет-соединение (рекомендуется проводное)

### Для сборки из исходников
- Go 1.21 или новее
- Git (опционально)

## Установка

### Вариант 1: Скачать готовый бинарник (Рекомендуется)

1. Скачайте последнюю версию для вашей платформы со страницы [Releases](../../releases)
2. Распакуйте архив в папку
3. Откройте терминал/командную строку в этой папке

### Вариант 2: Собрать из исходников

1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/yourusername/specter.git
   cd specter
   ```

2. Установите зависимости:
   ```bash
   go mod download
   ```

3. Соберите:
   ```bash
   # macOS/Linux
   go build -o specter

   # Windows
   go build -o specter.exe
   ```

## Первоначальная настройка

**ВАЖНО**: Завершите эту настройку ДО начала распродажи корабля!

### Шаг 1: Первый вход

1. Запустите приложение в первый раз:
   ```bash
   ./specter
   ```

2. Откроется окно браузера. Приложение покажет ошибку ("No item URL specified") - **это нормально**.

3. **В открывшемся окне браузера**: Войдите в свой аккаунт RSI

4. **Оставьте окно браузера открытым** и дайте приложению завершиться с ошибкой

5. Ваша сессия входа теперь сохранена!

### Шаг 2: Настройка целевого товара

1. Откройте `config.yaml` в текстовом редакторе

2. Установите URL корабля/предмета, который хотите купить:
   ```yaml
   item_url: "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"
   ```

3. **(Опционально)** Настройте параметры времени для максимальной скорости (см. [Советы для максимальной скорости](#советы-для-максимальной-скорости))

### Шаг 3: Тестовый запуск

**Сделайте тестовый запуск с обычным (не лимитированным) кораблем перед реальной распродажей!**

```bash
# Тест с режимом dry-run (не завершит покупку)
./specter -dry-run

# Или тест с дешевым предметом, который не жалко купить
./specter -url "https://robertsspaceindustries.com/pledge/Standalone-Ships/Aurora-ES"
```

Это проверит, что всё работает, и прогреет профиль браузера.

## Конфигурация

Файл `config.yaml` управляет всеми настройками. Конфигурация по умолчанию создается при первом запуске.

### Полный пример конфигурации

```yaml
# URL целевого товара
item_url: "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"

# Настройки браузера
browser_profile_path: /Users/username/.specter/browser-profile
browser_type: chrome                    # chrome, edge или firefox
headless: false                         # true = без видимого окна
keep_browser_open: true                 # Оставить браузер открытым после завершения

# Настройки времени (в секундах)
page_load_timeout: 30                   # Макс время ожидания загрузки страниц
min_delay_between: 0.5                  # Минимальная задержка между действиями
max_delay_between: 1.0                  # Максимальная задержка между действиями
checkout_ready_delay: 2                 # Задержка перед началом (теперь игнорируется в коде)

# Настройки повторов
max_retries: 30                         # Количество попыток повтора
retry_delay_min: 0.4                    # Мин задержка между повторами
retry_delay_max: 0.8                    # Макс задержка между повторами

# Флаги функций
auto_apply_credit: true                 # Автоматически применять store credit
dry_run: false                          # Тестовый режим (останавливается перед покупкой)
interactive: false                      # Пауза на каждом шаге для проверки
debug_mode: false                       # Включить подробное логирование

# Настройки дисплея
viewport_width: 1920
viewport_height: 1080

# CSS Селекторы (продвинутое - менять только если структура сайта изменилась)
selectors:
    add_to_cart_button: .m-storeAction__button
```

### Справочник параметров конфигурации

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `item_url` | string | "" | Прямой URL товара для покупки |
| `browser_profile_path` | string | ~/.specter/browser-profile | Расположение хранилища сессии браузера |
| `browser_type` | string | "chrome" | Браузер: chrome, edge или firefox |
| `headless` | bool | false | Запуск без видимого окна (немного быстрее) |
| `keep_browser_open` | bool | true | Оставить браузер открытым после завершения |
| `page_load_timeout` | int | 30 | Макс секунд ожидания страниц |
| `min_delay_between` | float | 0.5 | Мин секунд между действиями |
| `max_delay_between` | float | 1.0 | Макс секунд между действиями |
| `checkout_ready_delay` | int | 2 | Начальная задержка (устарело, теперь игнорируется) |
| `max_retries` | int | 30 | Попытки повтора при неудаче |
| `retry_delay_min` | float | 0.4 | Мин секунд между повторами |
| `retry_delay_max` | float | 0.8 | Макс секунд между повторами |
| `auto_apply_credit` | bool | true | Авто-применение store credit |
| `dry_run` | bool | false | Тестовый режим (останавливается перед покупкой) |
| `interactive` | bool | false | Пауза на каждом шаге |
| `debug_mode` | bool | false | Подробное логирование |
| `viewport_width` | int | 1920 | Ширина окна браузера |
| `viewport_height` | int | 1080 | Высота окна браузера |

## Параметры командной строки

Флаги командной строки переопределяют настройки из config файла.

### Доступные флаги

```bash
-config <путь>        # Путь к файлу конфигурации (по умолчанию: config.yaml)
-url <url>            # URL целевого товара (переопределяет config)
-dry-run              # Тестовый режим: останавливается перед кликом "I agree"
-interactive          # Пауза на каждом шаге для ручной проверки
-debug                # Включает подробное логирование отладки
-skip-cart            # Пропустить шаг добавления в корзину (товар уже в корзине)
```

### Подробности флагов

#### `-config <путь>`
Использовать пользовательский файл конфигурации. Полезно для управления несколькими целевыми товарами.

**Пример:**
```bash
./specter -config idris-config.yaml
```

#### `-url <url>`
Указать целевой URL напрямую без редактирования config файла.

**Пример:**
```bash
./specter -url "https://robertsspaceindustries.com/pledge/ships/anvil-carrack/Carrack"
```

#### `-dry-run`
Протестировать весь процесс без завершения покупки. Приложение:
- Перейдет к товару
- Добавит в корзину
- Пройдет через оформление
- Применит store credit
- Примет условия
- **Остановится перед кликом "I agree"** (финальная кнопка покупки)

**Пример:**
```bash
./specter -dry-run
```

**Используйте для**: Проверки, что всё работает перед реальной распродажей.

#### `-interactive`
Делает паузу на каждом шаге и ждет нажатия Enter. Полезно для:
- Понимания процесса
- Ручной проверки каждого шага
- Отладки проблем

**Пример:**
```bash
./specter -interactive
```

#### `-debug`
Включает подробное логирование, показывающее:
- Результаты выполнения JavaScript
- Информацию о времени
- Попытки поиска элементов
- Изменения внутреннего состояния

**Пример:**
```bash
./specter -debug
```

**Используйте для**: Диагностики проблем или понимания происходящего.

#### `-skip-cart`
Пропускает шаг "добавить в корзину" и идет прямо к оформлению. Используйте когда:
- Товар уже в вашей корзине с предыдущей попытки
- Вы вручную добавили товар и хотите использовать приложение только для оформления
- Повторная попытка после неудачи

**Пример:**
```bash
./specter -skip-cart
```

**Примечание**: Приложение перейдет напрямую на страницу корзины/оформления.

### Комбинирование флагов

Можно комбинировать несколько флагов:

```bash
# Тестовый запуск с логированием отладки
./specter -dry-run -debug

# Пропустить до оформления с пользовательским URL
./specter -skip-cart -url "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"

# Интерактивный режим с логированием отладки
./specter -interactive -debug

# Продакшн запуск с определенной конфигурацией
./specter -config production-config.yaml -url "https://robertsspaceindustries.com/pledge/ships/anvil-carrack/Carrack"
```

## Примеры использования

### Основные сценарии использования

#### 1. Первый тест (перед распродажей)
```bash
# Полный тест в режиме dry-run
./specter -dry-run -debug
```
Это проверит вашу настройку без завершения покупки.

#### 2. Продакшн запуск (во время распродажи)
```bash
# Использовать настройки из config файла
./specter

# Или указать URL напрямую
./specter -url "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"
```

#### 3. Повтор после неудачной попытки (товар всё ещё в корзине)
```bash
# Пропустить добавление в корзину и сразу к оформлению
./specter -skip-cart
```

#### 4. Несколько целевых товаров
Создайте отдельные config файлы:

**idris.yaml:**
```yaml
item_url: "https://robertsspaceindustries.com/pledge/ships/aegis-idris/Idris-M"
# ... другие настройки
```

**javelin.yaml:**
```yaml
item_url: "https://robertsspaceindustries.com/pledge/ships/aegis-javelin/Javelin"
# ... другие настройки
```

Затем запускайте:
```bash
./specter -config idris.yaml
# или
./specter -config javelin.yaml
```

#### 5. Диагностический запуск
```bash
# Увидеть точно что происходит
./specter -debug -interactive
```

### Чеклист перед распродажей

Завершите этот чеклист за 15-30 минут до распродажи:

```bash
# 1. Проверить, что вход всё ещё активен
./specter -dry-run

# 2. Если вход истек, откроется браузер - войдите снова

# 3. Сделать полный тестовый запуск
./specter -dry-run -debug

# 4. Проверить правильность конфигурации
cat config.yaml | grep item_url
cat config.yaml | grep dry_run  # Должно быть: false

# 5. Закрыть все другие приложения для освобождения ресурсов

# 6. Готовы к продакшну!
./specter
```

## Как это работает

### Полный процесс

1. **Запуск браузера** (1-2 секунды)
   - Запускает браузер с сохраненным профилем
   - Поддерживает вашу сессию входа

2. **Навигация** (100-500мс)
   - Переходит к URL целевого товара
   - Извлекает цену товара для проверки

3. **Добавление в корзину** (100-300мс)
   - Находит и кликает кнопку "Add to cart"
   - Ждет обновления корзины

4. **Переход в корзину** (300-500мс)
   - Идет на страницу корзины
   - Проверяет наличие товара

5. **Переход к оформлению** (400-600мс)
   - Кликает кнопку checkout
   - Ждет страницу оформления

6. **Применение Store Credit** (600-900мс если применимо)
   - Проверяет, уже ли итог $0
   - Если нет, кликает поле кредита и вводит сумму
   - Ждет применения кредита

7. **Переход к оплате** (переменно)
   - Проверяет, уже ли на шаге 2
   - Кликает "Proceed to pay" если нужно
   - Ждет модальное окно дисклеймера

8. **Принятие условий** (400-600мс)
   - Кликает "Jump to bottom" в модальном окне
   - Отмечает чекбокс соглашения
   - Ждет активации кнопки

9. **Финализация** (мгновенно)
   - Кликает кнопку "I agree"
   - Покупка завершена!

**Типичное общее время**: 3-6 секунд от старта до завершения (после запуска браузера)

### Логика повторов

Если оформление не удалось (товар недоступен, таймаут и т.д.):
- Автоматически повторяет до 30 раз (настраивается)
- Ждет 400-800мс между попытками
- При повторе возвращается к шагу 1 и пробует снова
- Отслеживает, уже ли товар в корзине, чтобы пропустить шаг 3 при повторах

## Советы для максимальной скорости

### Критичные настройки для скорости

Отредактируйте ваш `config.yaml`:

```yaml
# Абсолютный минимум задержек (используйте на свой риск)
min_delay_between: 0.3
max_delay_between: 0.5
retry_delay_min: 0.2
retry_delay_max: 0.4

# Отключить ненужные функции
keep_browser_open: false
debug_mode: false

# Опционально: немного быстрее (без видимого окна)
headless: true
```

### Чеклист оптимизации скорости

**Перед распродажей:**
- ✅ Используйте проводное ethernet соединение (не WiFi)
- ✅ Закройте все другие приложения
- ✅ Закройте все другие окна/вкладки браузера
- ✅ Временно отключите антивирус (если комфортно)
- ✅ Используйте headless режим: `headless: true`
- ✅ Снизьте все значения времени до минимума
- ✅ Установите `keep_browser_open: false`
- ✅ Увеличьте max_retries: `max_retries: 50`

**Выбор браузера:**
- Chrome самый быстрый и надежный
- Edge сравним с Chrome
- Firefox немного медленнее

**Система:**
- Закройте Discord, Slack, Steam и т.д.
- Отключите системные уведомления
- Освободите RAM (закройте ненужные приложения)

### Сбалансированные vs Агрессивные настройки

**Сбалансированные (Рекомендуется):**
```yaml
min_delay_between: 0.5
max_delay_between: 1.0
max_retries: 30
headless: false
```
- Надежно и достаточно быстро
- Низкий риск обнаружения бота
- Хорошо для большинства распродаж

**Агрессивные (Максимальная скорость):**
```yaml
min_delay_between: 0.2
max_delay_between: 0.4
max_retries: 50
headless: true
```
- Абсолютно максимально быстро
- Повышенный риск обнаружения бота (используйте осторожно)
- Только для высококонкурентных распродаж

## Устранение неполадок

### Проблемы с браузером

#### "Failed to get the debug url: Opening in existing browser session"
**Причина**: Другой экземпляр использует профиль браузера.

**Решение**:
```bash
# Убить все запущенные экземпляры
pkill -f specter

# На Windows, используйте Диспетчер задач чтобы завершить specter.exe

# Затем запустите снова
./specter
```

#### "Browser launch failed"
**Причина**: Браузер не найден или профиль поврежден.

**Решения**:
1. Убедитесь что Chrome/Edge/Firefox установлен
2. Попробуйте удалить профиль браузера:
   ```bash
   # macOS/Linux
   rm -rf ~/.specter/browser-profile

   # Windows
   rmdir /s %USERPROFILE%\.specter\browser-profile
   ```
3. Запустите снова и войдите

### Проблемы с оформлением

#### "Could not find add to cart button"
**Причина**: HTML сайта изменился или товар недоступен.

**Решения**:
1. Проверьте, действительно ли товар в продаже
2. Проверьте правильность URL
3. Попробуйте обновить селектор в config:
   ```yaml
   selectors:
       add_to_cart_button: ".your-custom-selector"
   ```
4. Запустите с `-debug` чтобы увидеть что происходит

#### "Failed to apply store credit"
**Причина**: Поле кредита не найдено или итог уже $0.

**Решения**:
1. Проверьте наличие store credit в вашем аккаунте
2. Проверьте, не $0 ли цена товара (кредит мог уже быть применен)
3. Установите `auto_apply_credit: false` и примените вручную

#### Приложение останавливается на странице оформления
**Причина**: Модальное окно дисклеймера не появляется или проблема с таймингом.

**Решения**:
1. Запустите с `-interactive` для ручного прохождения шагов
2. Проверьте, действительно ли вы вошли в систему
3. Увеличьте page_load_timeout: `page_load_timeout: 60`
4. Используйте `-debug` для просмотра подробных логов

### Проблемы со скоростью

#### Приложение кажется медленным
**Причины и решения**:
1. Проверьте значения времени в config - уменьшите их
2. Убедитесь в быстром интернет-соединении
3. Используйте `headless: true` для небольшого ускорения
4. Закройте другие приложения
5. Проверьте, включены ли расширения браузера (они замедляют)

#### Повторы происходят немедленно
**Ожидаемое поведение**: Приложение автоматически повторяет при неудаче оформления.

**Чтобы увидеть почему происходят повторы**:
```bash
./specter -debug
```
Ищите сообщения об ошибках, показывающие почему оформление не удалось.

### Проблемы со входом

#### Приложение не помнит вход
**Причина**: Профиль браузера не сохранен или поврежден.

**Решение**:
1. Проверьте browser_profile_path в config
2. Убедитесь что директория существует и доступна для записи
3. Войдите снова и позвольте браузеру сохранить сессию
4. Проверьте что `~/.specter/browser-profile` содержит файлы

#### Сессия истекла во время распродажи
**Профилактика**:
- Войдите за 15 минут до распродажи
- Сделайте тестовый запуск для прогрева сессии
- Держите профиль браузера активным

### Режим отладки

Для любой необъяснимой проблемы, запустите с полной отладкой:

```bash
./specter -debug -interactive
```

Это:
- Покажет все внутренние операции
- Сделает паузу на каждом шаге
- Отобразит результаты выполнения JavaScript
- Покажет информацию о времени

Поделитесь выводом отладки при обращении за помощью.

## Расположение профиля браузера

Ваша сессия входа хранится в:

**macOS/Linux:**
```
~/.specter/browser-profile
```

**Windows:**
```
%USERPROFILE%\.specter\browser-profile
```

**Для сброса (выхода)**:
```bash
# macOS/Linux
rm -rf ~/.specter/browser-profile

# Windows (Командная строка)
rmdir /s %USERPROFILE%\.specter\browser-profile

# Windows (PowerShell)
Remove-Item -Recurse -Force $env:USERPROFILE\.specter\browser-profile
```

## Правовая оговорка

⚠️ **ВАЖНО**: Пожалуйста, внимательно прочитайте.

Этот инструмент предназначен **только для личного использования**. Использование инструментов автоматизации может нарушать Условия использования robertsspaceindustries.com. Используйте на свой риск.

**Используя это программное обеспечение, вы подтверждаете что**:
- Вы несете единоличную ответственность за использование этого инструмента
- Автоматизированная покупка может дать несправедливое преимущество перед другими пользователями
- Вы должны ознакомиться и соблюдать Условия использования RSI
- Авторы не несут ответственности за любые последствия, включая приостановку или прекращение аккаунта
- Это программное обеспечение предоставляется "как есть" без каких-либо гарантий

**Нет сбора данных**: Это приложение не собирает, не хранит и не передает никакие персональные данные. Всё работает локально на вашей машине.

**Безопасность**: Ваши учетные данные RSI обрабатываются только самим браузером, никогда этим приложением. Приложение только отправляет команды автоматизации браузера.

## Сборка из исходников

### Зависимости

```bash
go mod download
```

### Команды сборки

```bash
# Текущая платформа
go build -o specter

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o specter-darwin-amd64

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o specter-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o specter.exe

# Linux
GOOS=linux GOARCH=amd64 go build -o specter-linux
```

## Содействие

Приветствуются вклады! Пожалуйста:
1. Форкните репозиторий
2. Создайте ветку для функции
3. Внесите изменения
4. Тщательно протестируйте
5. Отправьте pull request

## Поддержка

- 🐛 **Проблемы**: Откройте issue на GitHub
- 💬 **Вопросы**: Сначала проверьте существующие issues
- 📖 **Документация**: Этот README

## Лицензия

Лицензия MIT - см. файл LICENSE для деталей

## Благодарности

- Создано с [Rod](https://github.com/go-rod/rod) - Высокопроизводительная автоматизация браузера
- Вдохновлено необходимостью справедливого доступа к товарам ограниченной доступности

---

**Помните / Remember**: Этот инструмент предназначен для выравнивания игрового поля против других ботов. Пожалуйста, используйте ответственно и в соответствии с Условиями использования RSI. / This tool is meant to level the playing field against other bots. Please use responsibly and in accordance with RSI's Terms of Service.
