# Specter - RSI Store Automated Checkout

**Lightning-fast automated checkout for limited Star Citizen ship sales**

[English](#english) | [Русский](#русский)

---

## English

### What is This?

Specter is a tool that automatically buys limited-edition ships from the Star Citizen store (robertsspaceindustries.com) at lightning speed using **store credit only**. When ships sell out in seconds, this gives you the best chance to complete your purchase across multiple sale waves.

**⚠️ IMPORTANT LIMITATIONS:**
- ✅ **Works ONLY with store credit** - Cannot process credit card or PayPal payments
- ✅ **Single ship purchases only** - Designed for buying one ship at a time
- ❌ Does NOT work for cash/credit card purchases
- ❌ Does NOT work for game packages or multi-item purchases

**Key Features:**
- 🌊 **Multi-Wave Automated Mode** - Set up all sale waves at once, app handles everything automatically
- ⚡ **Ultra-fast checkout** - Completes purchase in under 1 second once item is in cart
- 🕐 **Time Synchronization** - Syncs with network time servers for perfect timing accuracy
- 📅 **User-Friendly Time Format** - Just copy "2025-01-15 16:00 UTC" from CIG website and paste it
- 🔄 **Aggressive retry system** - Smart backoff when servers are busy
- 💳 **Automatic store credit application** - No manual steps needed
- 🛡️ **Cart validation safeguards** - Prevents accidentally buying multiple items or wrong ships
- 🎯 **Optimized for speed** - Every millisecond counts when competing for limited ships
- 🌍 **Multi-language support** - Automatically detects your system language (English, Russian supported)

### Requirements

**What You Need:**
- A computer (Windows 10/11 or Mac)
- Google Chrome browser installed (strongly recommended - avoids download issues)
- **A Star Citizen account with SUFFICIENT store credit** - The app ONLY works with store credit payments
- The ship must be purchasable as a single standalone item (not a package)
- Basic computer skills (opening files, running programs)

**⚠️ CRITICAL:** You must have enough store credit in your RSI account to cover the full price of the ship. The app cannot use credit cards, PayPal, or any other payment method.

**Note:** Specter will automatically use your installed Chrome browser if available. If Chrome is not installed, it will download a temporary browser (may require antivirus exclusions on Windows).

### Installation

#### For Windows:

1. **Download Specter:**
   - Go to: **https://github.com/aconcernedstarcitizen/spectre/releases**
   - Download the latest `specter-windows-amd64.zip` file (look for "Assets" section)
   - **Right-click the ZIP file** and select "Extract All..."
   - Extract to a folder (like `C:\Specter`)
   - The extracted folder will contain:
     - `specter.exe` - The program
     - `config.yaml` - Configuration file
     - `lang/` - Language files (auto-detects your system language)

2. **Make sure Chrome is installed:**
   - If you don't have Chrome, download it from google.com/chrome

#### For Mac:

1. **Download Specter:**
   - Go to: **https://github.com/aconcernedstarcitizen/spectre/releases**
   - Download the latest ZIP file for Mac:
     - `specter-macos-arm64.zip` if you have Apple Silicon (M1/M2/M3/M4)
     - `specter-macos-amd64.zip` if you have an Intel Mac
   - **Double-click the ZIP file** to extract it
   - Move the extracted folder to a location like `/Users/YourName/Specter`
   - The extracted folder will contain:
     - `specter` - The program
     - `config.yaml` - Configuration file
     - `lang/` - Language files (auto-detects your system language)

2. **Make it runnable:**
   - Open Terminal (search for "Terminal" in Spotlight)
   - Type: `cd ` (with a space at the end)
   - Drag the Specter folder into Terminal and press Enter
   - Type: `chmod +x specter` and press Enter

3. **Make sure Chrome is installed:**
   - If you don't have Chrome, download it from google.com/chrome

### Setup (Do This Before the Sale!)

**IMPORTANT: Complete these steps at least 30 minutes before the ship sale!**

#### Step 1: Configure Sale Windows

1. **Open the Specter folder** where you extracted the ZIP file

2. **Find and open `config.yaml`** with a text editor:
   - **Windows:** Right-click `config.yaml` → Open with → Notepad
   - **Mac:** Right-click `config.yaml` → Open With → TextEdit

3. **Find the section that says `sale_windows:`** (around line 69)

4. **Add your sale times** - Copy the times from the CIG website and paste them.

   **Standard CIG wave times**:
    ```yaml
    sale_windows:
    # IAE 2955 example: Constellation Phoenix
    - "2025-11-20 16:00"
    - "2025-11-20 20:00"
    - "2025-11-21 00:00"
    - "2025-11-21 04:00"
    - "2025-11-21 08:00"
    - "2025-11-21 12:00"
    # IAE 2955 example: 890 Jump
    - "2025-11-21 16:00"
    - "2025-11-21 20:00"
    - "2025-11-22 00:00"
    - "2025-11-22 04:00"
    - "2025-11-22 08:00"
    - "2025-11-22 12:00"
    # IAE 2955 example: Kraken, Kraken Privateer
    - "2025-11-22 16:00"
    - "2025-11-22 20:00"
    - "2025-11-23 00:00"
    - "2025-11-23 04:00"
    - "2025-11-23 08:00"
    - "2025-11-23 12:00"
    # IAE 2955 example: Hull E
    - "2025-11-24 16:00"
    - "2025-11-24 20:00"
    - "2025-11-25 00:00"
    - "2025-11-25 04:00"
    - "2025-11-25 08:00"
    - "2025-11-25 12:00"
    # IAE 2955 example: Pioneer
    - "2025-11-26 16:00"
    - "2025-11-26 20:00"
    - "2025-11-27 00:00"
    - "2025-11-27 04:00"
    - "2025-11-27 08:00"
    - "2025-11-27 12:00"
    # IAE 2955 example: Idris-P, Javelin
    - "2025-11-28 16:00"
    - "2025-11-28 20:00"
    - "2025-11-29 00:00"
    - "2025-11-29 04:00"
    - "2025-11-29 08:00"
    - "2025-11-29 12:00"
    ```

   **Important:**
   - Times MUST be in UTC timezone (check CIG's announcements)
   - Format: `"YYYY-MM-DD HH:MM"` (24-hour format)
   - You can add as many or as few waves as you want
   - Remove the `#` comments if you copy this example

5. **Set the ship URL** - Find the line that says `item_url:` (around line 11)

   ```yaml
   item_url: "https://robertsspaceindustries.com/pledge/ships/anvil-carrack/Carrack"
   ```

   Replace the URL with the exact URL of the ship you want to buy.

6. **Save the file** (File → Save)

#### Step 2: First-Time Login

**This creates a saved browser session so you don't need to log in during the sale:**

1. **For Windows:**
   - Open Command Prompt (search for "cmd" in Start menu)
   - Type: `cd C:\Specter` (or wherever you saved it)
   - Type: `specter.exe`
   - Press Enter

   **For Mac:**
   - Open Terminal (search for "Terminal" in Spotlight)
   - Type: `cd ` (with space)
   - Drag the Specter folder into Terminal and press Enter
   - Type: `./specter`
   - Press Enter

2. **You'll see an error message** saying "No sale windows configured" - **THIS IS EXPECTED!**
   - This happens because we haven't added sale times yet (we'll do that in Step 1)
   - But the important part is that a Chrome window opened

3. **In the Chrome window that opened:**
   - Go to robertsspaceindustries.com
   - **Log in to your RSI account**
   - Make sure you see your username in the top right

4. **Close the Chrome window** - Your login is now saved!

#### Step 3: Test It! (CRITICAL - DO NOT SKIP)

**Always test before the real sale to make sure everything works!**

**Windows:**
```
cd C:\Specter
specter.exe --dry-run
```

**Mac:**
```
cd /Users/YourName/Specter
./specter --dry-run
```

**What should happen:**
1. Chrome opens (you should already be logged in)
2. App shows your configured waves
3. Program waits for the first wave
4. Program will say it would attempt checkout, but stops before actually buying (because of `--dry-run`)

If you see errors, fix them now! Common issues:
- "No sale windows configured" → Go back to Step 1 and add sale times
- "No item URL specified" → Go back to Step 1 and add the ship URL
- Not logged in → Go back to Step 2

### How to Use - Multi-Wave Mode

**You can start the app at ANY time - it will automatically figure out which wave to process!**

**Windows:**
```
cd C:\Specter
specter.exe
```

**Mac:**
```
cd /Users/YourName/Specter
./specter
```

**What happens automatically:**

1. **Time Synchronization** (first few seconds)
   ```
   🔄 Synchronizing time with Amazon time server...
   ✓ Time synchronized (system clock is 234ms behind network time)
   ```
   - Syncs with Amazon's time server (since CIG hosts infrastructure on AWS)
   - Calculates precise time offset for accurate wave timing

2. **Wave Schedule Display & Smart Wave Selection**
   ```
   🌊 Multi-Wave Automated Mode Enabled
      📅 Configured waves: 6
      ⏰ Pre-wave activation: 2 minutes before each wave
      ⏱️  Post-wave timeout: 5 minutes after each wave

      Wave schedule:
     Wave 1: 2025-01-15 16:00 UTC (11:00:00 EST)
     Wave 2: 2025-01-15 20:00 UTC (15:00:00 EST)
     Wave 3: 2025-01-16 00:00 UTC (19:00:00 EST)
     ...
   ```

   **The app is SMART - it knows what time it is:**

   **Scenario A: Started BEFORE first wave**
   ```
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   🌊 WAVE 1 of 6
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ```
   → App waits for Wave 1

   **Scenario B: Started AFTER Wave 1 ended (e.g., 16:08 UTC)**
   ```
   ⏩ Skipping 1 past wave(s)...
      • Wave 1 (11:00:00 EST) - Ended

   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   🌊 WAVE 2 of 6
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ```
   → App automatically skips Wave 1 and goes to Wave 2

   **Scenario C: Started AFTER all waves ended**
   ```
   ⚠️  All sale waves have already ended!
      Last wave (Wave 6) ended at: 12:05:00 EST
   👋 Exiting - no active or upcoming waves remaining
   ```
   → App informs you and exits gracefully

3. **Login Prompt** (if not already logged in)
   ```
   ============================================================
                     LOGIN REQUIRED
   ============================================================

   Please log in to your RSI account in the browser window.

   Press ENTER when ready...
   ```
   - Chrome opens automatically
   - If you're already logged in (from Step 2), just press ENTER
   - If not, log in now, then press ENTER

4. **Waiting for Next Wave**
   ```
   ⏳ Waiting 2h 15m 30s until pre-wave activation...
      Activation at: 15:58:00 EST
   ```
   - App sleeps until 2 minutes before the wave
   - You can leave your computer - app will wake up automatically
   - Progress updates every 30 seconds

5. **Pre-Wave Polling** (2 minutes before wave)
   ```
   🔍 Pre-wave polling started - checking product page availability...
      Polling: https://robertsspaceindustries.com/pledge/ships/...

      Status 404 - Wave starts in 1m 45s
      Status 404 - Wave starts in 1m 35s
   ```
   - App checks every second if the product page is available
   - When it changes from 404 to 200, it means the sale is live!

6. **Product Available!**
   ```
   ✅ Product page is now available!
      (Page went live 15s before scheduled time)

   📄 Navigating to product page...
   🔍 Extracting SKU from product page...
   ```

7. **Checkout Attempt**
   ```
   🚀 Attempting checkout...
      Will timeout at: 16:05:00 EST

   🔍 Checking current cart state...
   ✓ Cart is empty, will add item
   🛒 Adding to cart (API) with retry mechanism...
   🔄 Attempt 1 - Time remaining: 4m59s
   🔄 Attempt 87 - Time remaining: 4m57s
   ✅ Successfully added to cart after 87 attempts in 2.3s!

   💰 Applying store credit...
   ➡️  Moving to billing/addresses step...
   📍 Assigning billing address...
   🎯 Completing order...

   ✅ Purchase successful!
   👋 Exiting multi-wave mode (checkout completed successfully)
   ```
   - If successful: **App exits gracefully** - YOU GOT THE SHIP!
   - If failed: App continues trying for 5 more minutes, then moves to next wave

8. **If Wave Fails** (automatically stays dormant until next wave)
   ```
   ❌ Wave 2: Checkout unsuccessful
   ➡️  Moving to Wave 3...
      Next wave starts in: 3h 52m 18s
   💤 Staying dormant until next wave activation time
   ```
   - App waits patiently for the next wave
   - You don't need to restart the app
   - Process repeats for all remaining waves

9. **If Last Wave Fails**
   ```
   ❌ Wave 6: Checkout unsuccessful
      This was the last wave - no more waves remaining

   ❌ All waves completed without successful checkout
   ```
   - App informs you all waves have been attempted
   - Unfortunately the ship sold out

### Advanced Options

You can customize wave timing with command-line flags:

**Change pre-wave polling time** (default: 2 minutes before)
```
specter.exe --pre-wave 3
```
This starts checking for the product page 3 minutes before each wave instead of 2.

**Change post-wave timeout** (default: 5 minutes after)
```
specter.exe --post-wave 10
```
This keeps trying for 10 minutes after each wave before moving to the next.

**Combine both:**
```
specter.exe --pre-wave 3 --post-wave 10
```

**Debug mode** (see detailed technical information):
```
specter.exe --debug
```

### Troubleshooting

**"No sale windows configured"**
- You forgot to add sale times in config.yaml
- Go back to Setup Step 1

**"All sale waves have already ended!"**
- You started the app after all waves finished
- The last wave ended at the time shown on screen
- Check RSI's Spectrum/Discord for potential additional waves or future sales

**"Product page never became available (timed out)"**
- The ship sale might be cancelled or delayed
- Check RSI's website or Discord for updates
- Your sale times might be wrong (check timezone - must be UTC!)

**"All waves completed without successful checkout"**
- The ship sold out in all waves before you could get it
- Unfortunately this means it's gone
- Check Spectrum/Discord for potential additional waves

**"Browser is not responding"**
- Close all Chrome windows completely
- On Windows: Open Task Manager → End all Chrome processes
- On Mac: Command+Q to quit Chrome, or Activity Monitor → Force Quit Chrome
- Try running Specter again

**Windows Defender blocks the program**
- This is a false positive (common with new programs)
- Click "More info" → "Run anyway"
- Or add an exclusion: Windows Security → Virus & threat protection → Manage settings → Add exclusion → Choose the Specter folder

**Mac says "cannot be opened because it is from an unidentified developer"**
- Right-click `specter` → Open
- Click "Open" in the security warning
- Or: System Settings → Privacy & Security → Allow apps from: App Store and identified developers

**Still having issues?**
- Make sure you completed Step 2 (First-Time Login) successfully
- Make sure Chrome is installed and up to date
- Try the `--debug` flag to see detailed error messages

### Tips for Success

1. **Set up EARLY** - Do Step 1 and 2 at least 30 minutes before the sale
2. **Test with --dry-run** - Always test before the real sale!
3. **Check your store credit** - Make absolutely sure you have enough
4. **Use standard CIG wave times** - Double-check the times on Spectrum/Discord
5. **Don't touch the computer during waves** - Let the app work automatically
6. **Keep browser window visible** - Don't minimize it (can cause issues on some systems)
7. **Good internet connection** - Use wired Ethernet if possible

### How It Works (Technical Details)

For those interested in the technical implementation:

1. **Time Synchronization**: App uses HTTP HEAD requests to Amazon's time server to calculate precise time offset (CIG infrastructure runs on AWS for accurate timing)
2. **Smart Wave Detection**: On startup, compares current time against all wave end times (wave_time + post_wave_timeout) to determine which wave to start from
3. **Past Wave Skipping**: Automatically skips waves that have already ended, displays list of skipped waves to user
4. **Pre-Wave Polling**: Starting 2 minutes before each wave, sends HTTP HEAD requests every second checking for 200 status (product available)
5. **SKU Extraction**: Once page is available, uses browser JavaScript evaluation to extract SKU from multiple sources (Next.js data, script tags, component props)
6. **API-Based Checkout**: Bypasses browser UI entirely, sends direct GraphQL mutations to RSI's store API
7. **Smart Retry**: Implements exponential backoff for rate limits, specific delays for different error types (4226, 4227, out of stock, etc.)
8. **Cart Validation**: Detects if cart already has correct item with credits applied, skips redundant steps
9. **Address Caching**: Pre-fetches and caches billing address to eliminate lookup delays during checkout
10. **reCAPTCHA v3**: Generates fresh Enterprise tokens for each cart addition attempt
11. **Multi-Wave State Machine**: Automatically transitions between waves on timeout, stays dormant between waves, exits gracefully on success or when all waves complete

---

## Русский

### Что это?

Specter - это инструмент, который автоматически покупает корабли ограниченного тиража из магазина Star Citizen (robertsspaceindustries.com) с молниеносной скоростью, используя **только внутримагазинный кредит**. Когда корабли распродаются за секунды, это дает вам наилучший шанс завершить покупку во время нескольких волн продаж.

**⚠️ ВАЖНЫЕ ОГРАНИЧЕНИЯ:**
- ✅ **Работает ТОЛЬКО с внутримагазинным кредитом** - Не может обрабатывать платежи кредитной картой или PayPal
- ✅ **Только покупка одного корабля** - Разработано для покупки одного корабля за раз
- ❌ НЕ работает для покупок за наличные/кредитную карту
- ❌ НЕ работает для игровых пакетов или покупок нескольких предметов

**Ключевые Функции:**
- 🌊 **Автоматический мультиволновый режим** - Настройте все волны продаж сразу, приложение все сделает автоматически
- ⚡ **Сверхбыстрая оформление заказа** - Завершает покупку менее чем за 1 секунду после добавления товара в корзину
- 🕐 **Синхронизация времени** - Синхронизируется с сетевыми серверами времени для идеальной точности
- 📅 **Удобный формат времени** - Просто скопируйте "2025-01-15 16:00 UTC" с сайта CIG и вставьте
- 🔄 **Агрессивная система повторных попыток** - Умная задержка при занятых серверах
- 💳 **Автоматическое применение кредита магазина** - Не требуется ручных действий
- 🛡️ **Защита проверки корзины** - Предотвращает случайную покупку нескольких предметов или не тех кораблей
- 🎯 **Оптимизировано для скорости** - Каждая миллисекунда важна при конкуренции за ограниченные корабли
- 🌍 **Многоязычная поддержка** - Автоматически определяет язык вашей системы (поддерживается английский, русский)

### Требования

**Что вам нужно:**
- Компьютер (Windows 10/11 или Mac)
- Установленный браузер Google Chrome (настоятельно рекомендуется - избегает проблем с загрузкой)
- **Учетная запись Star Citizen с ДОСТАТОЧНЫМ внутримагазинным кредитом** - Приложение работает ТОЛЬКО с платежами кредитом магазина
- Корабль должен быть доступен для покупки как отдельный предмет (не пакет)
- Базовые компьютерные навыки (открытие файлов, запуск программ)

**⚠️ КРИТИЧНО:** У вас должно быть достаточно внутримагазинного кредита на вашей учетной записи RSI, чтобы покрыть полную стоимость корабля. Приложение не может использовать кредитные карты, PayPal или любой другой способ оплаты.

**Примечание:** Specter автоматически будет использовать ваш установленный браузер Chrome, если он доступен. Если Chrome не установлен, будет загружен временный браузер (может потребоваться добавление исключений в антивирус на Windows).

### Установка

#### Для Windows:

1. **Скачайте Specter:**
   - Перейдите на: **https://github.com/aconcernedstarcitizen/spectre/releases**
   - Скачайте последний файл `specter-windows-amd64.zip` (ищите раздел "Assets")
   - **Щелкните правой кнопкой мыши на ZIP файле** и выберите "Извлечь все..."
   - Извлеките в папку (например, `C:\Specter`)
   - Извлеченная папка будет содержать:
     - `specter.exe` - Программа
     - `config.yaml` - Файл конфигурации
     - `lang/` - Языковые файлы (автоматически определяет язык вашей системы)

2. **Убедитесь, что Chrome установлен:**
   - Если у вас нет Chrome, загрузите его с google.com/chrome

#### Для Mac:

1. **Скачайте Specter:**
   - Перейдите на: **https://github.com/aconcernedstarcitizen/spectre/releases**
   - Скачайте последний ZIP файл для Mac:
     - `specter-macos-arm64.zip` если у вас Apple Silicon (M1/M2/M3/M4)
     - `specter-macos-amd64.zip` если у вас Intel Mac
   - **Дважды щелкните на ZIP файле** чтобы извлечь его
   - Переместите извлеченную папку в место вроде `/Users/ВашеИмя/Specter`
   - Извлеченная папка будет содержать:
     - `specter` - Программа
     - `config.yaml` - Файл конфигурации
     - `lang/` - Языковые файлы (автоматически определяет язык вашей системы)

2. **Сделайте его исполняемым:**
   - Откройте Terminal (ищите "Terminal" в Spotlight)
   - Введите: `cd ` (с пробелом в конце)
   - Перетащите папку Specter в Terminal и нажмите Enter
   - Введите: `chmod +x specter` и нажмите Enter

3. **Убедитесь, что Chrome установлен:**
   - Если у вас нет Chrome, загрузите его с google.com/chrome

### Настройка (Сделайте это до продажи!)

**ВАЖНО: Выполните эти шаги как минимум за 30 минут до продажи корабля!**

#### Шаг 1: Настройка окон продаж

1. **Откройте папку Specter**, где вы извлекли ZIP файл

2. **Найдите и откройте `config.yaml`** текстовым редактором:
   - **Windows:** Щелкните правой кнопкой `config.yaml` → Открыть с помощью → Блокнот
   - **Mac:** Щелкните правой кнопкой `config.yaml` → Открыть с помощью → TextEdit

3. **Найдите раздел `sale_windows:`** (около строки 69)

4. **Добавьте времена продаж** - Скопируйте времена с сайта CIG и вставьте их.

   **Стандартные времена волн CIG** (пример с IAE 2955):
    ```yaml
    sale_windows:
    # IAE 2955 example: Constellation Phoenix
    - "2025-11-20 16:00"
    - "2025-11-20 20:00"
    - "2025-11-21 00:00"
    - "2025-11-21 04:00"
    - "2025-11-21 08:00"
    - "2025-11-21 12:00"
    # IAE 2955 example: 890 Jump
    - "2025-11-21 16:00"
    - "2025-11-21 20:00"
    - "2025-11-22 00:00"
    - "2025-11-22 04:00"
    - "2025-11-22 08:00"
    - "2025-11-22 12:00"
    # IAE 2955 example: Kraken, Kraken Privateer
    - "2025-11-22 16:00"
    - "2025-11-22 20:00"
    - "2025-11-23 00:00"
    - "2025-11-23 04:00"
    - "2025-11-23 08:00"
    - "2025-11-23 12:00"
    # IAE 2955 example: Hull E
    - "2025-11-24 16:00"
    - "2025-11-24 20:00"
    - "2025-11-25 00:00"
    - "2025-11-25 04:00"
    - "2025-11-25 08:00"
    - "2025-11-25 12:00"
    # IAE 2955 example: Pioneer
    - "2025-11-26 16:00"
    - "2025-11-26 20:00"
    - "2025-11-27 00:00"
    - "2025-11-27 04:00"
    - "2025-11-27 08:00"
    - "2025-11-27 12:00"
    # IAE 2955 example: Idris-P, Javelin
    - "2025-11-28 16:00"
    - "2025-11-28 20:00"
    - "2025-11-29 00:00"
    - "2025-11-29 04:00"
    - "2025-11-29 08:00"
    - "2025-11-29 12:00"
    ```

   **Важно:**
   - Время ДОЛЖНО быть в часовом поясе UTC (проверьте объявления CIG)
   - Формат: `"YYYY-MM-DD HH:MM"` (24-часовой формат)
   - Вы можете добавить столько или столько мало волн, сколько захотите
   - Удалите комментарии `#` если копируете этот пример

5. **Установите URL корабля** - Найдите строку `item_url:` (около строки 11)

   ```yaml
   item_url: "https://robertsspaceindustries.com/pledge/ships/anvil-carrack/Carrack"
   ```

   Замените URL на точный URL корабля, который вы хотите купить.

6. **Сохраните файл** (Файл → Сохранить)

#### Шаг 2: Первоначальный вход

**Это создает сохраненную сессию браузера, чтобы вам не нужно было входить во время продажи:**

1. **Для Windows:**
   - Откройте Командную строку (ищите "cmd" в меню Пуск)
   - Введите: `cd C:\Specter` (или куда вы его сохранили)
   - Введите: `specter.exe`
   - Нажмите Enter

   **Для Mac:**
   - Откройте Terminal (ищите "Terminal" в Spotlight)
   - Введите: `cd ` (с пробелом)
   - Перетащите папку Specter в Terminal и нажмите Enter
   - Введите: `./specter`
   - Нажмите Enter

2. **Вы увидите сообщение об ошибке** "No sale windows configured" - **ЭТО ОЖИДАЕТСЯ!**
   - Это происходит потому что мы еще не добавили времена продаж (мы сделаем это в Шаге 1)
   - Но важная часть - что окно Chrome открылось

3. **В открывшемся окне Chrome:**
   - Перейдите на robertsspaceindustries.com
   - **Войдите в свою учетную запись RSI**
   - Убедитесь, что видите свое имя пользователя в правом верхнем углу

4. **Закройте окно Chrome** - Ваш вход теперь сохранен!

#### Шаг 3: Протестируйте! (КРИТИЧНО - НЕ ПРОПУСКАЙТЕ)

**Всегда тестируйте перед настоящей продажей, чтобы убедиться, что все работает!**

**Windows:**
```
cd C:\Specter
specter.exe --dry-run
```

**Mac:**
```
cd /Users/ВашеИмя/Specter
./specter --dry-run
```

**Что должно произойти:**
1. Открывается Chrome (вы уже должны быть в системе)
2. Приложение показывает ваши настроенные волны
3. Программа ждет первой волны
4. Программа скажет, что попыталась бы оформить заказ, но останавливается перед фактической покупкой (из-за `--dry-run`)

Если вы видите ошибки, исправьте их сейчас! Частые проблемы:
- "No sale windows configured" → Вернитесь к Шагу 1 и добавьте времена продаж
- "No item URL specified" → Вернитесь к Шагу 1 и добавьте URL корабля
- Не вошли в систему → Вернитесь к Шагу 2

### Как использовать - Мультиволновый режим

**Вы можете запустить приложение в ЛЮБОЕ время - оно автоматически определит, какую волну обрабатывать!**

**Windows:**
```
cd C:\Specter
specter.exe
```

**Mac:**
```
cd /Users/ВашеИмя/Specter
./specter
```

**Что происходит автоматически:**

1. **Синхронизация времени** (первые несколько секунд)
   ```
   🔄 Синхронизация времени с сервером времени Amazon...
   ✓ Время синхронизировано (системные часы на 234мс отстают от сетевого времени)
   ```
   - Синхронизация с сервером времени Amazon (так как инфраструктура CIG размещена на AWS)
   - Вычисляет точное смещение времени для точного таймирования волн

2. **Отображение расписания волн и умный выбор волны**
   ```
   🌊 Включен режим автоматических мультиволновых продаж
      📅 Настроено волн: 6
      ⏰ Активация перед волной: за 2 минуты до каждой волны
      ⏱️  Таймаут после волны: 5 минут после каждой волны

      Расписание волн:
     Волна 1: 2025-01-15 16:00 UTC (19:00:00 МСК)
     Волна 2: 2025-01-15 20:00 UTC (23:00:00 МСК)
     Волна 3: 2025-01-16 00:00 UTC (03:00:00 МСК следующий день)
     ...
   ```

   **Приложение УМНОЕ - оно знает, который час:**

   **Сценарий A: Запущено ДО первой волны**
   ```
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   🌊 ВОЛНА 1 из 6
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ```
   → Приложение ждет Волну 1

   **Сценарий B: Запущено ПОСЛЕ окончания Волны 1 (например, в 16:08 UTC)**
   ```
   ⏩ Пропуск 1 прошедших волн...
      • Волна 1 (19:00:00 МСК) - Завершена

   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   🌊 ВОЛНА 2 из 6
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ```
   → Приложение автоматически пропускает Волну 1 и переходит к Волне 2

   **Сценарий C: Запущено ПОСЛЕ окончания всех волн**
   ```
   ⚠️  Все волны распродажи уже завершились!
      Последняя волна (Волна 6) закончилась в: 15:05:00 МСК
   👋 Выход - активных или предстоящих волн не осталось
   ```
   → Приложение информирует вас и корректно завершает работу

3. **Запрос на вход** (если еще не вошли)
   ```
   ============================================================
                     ТРЕБУЕТСЯ ВХОД В СИСТЕМУ
   ============================================================

   Пожалуйста, войдите в свою учетную запись RSI в окне браузера.

   Нажмите ENTER когда будете готовы...
   ```
   - Chrome открывается автоматически
   - Если вы уже вошли (из Шага 2), просто нажмите ENTER
   - Если нет, войдите сейчас, затем нажмите ENTER

4. **Ожидание следующей волны**
   ```
   ⏳ Ожидание 2ч 15м 30с до активации перед волной...
      Активация в: 18:58:00 МСК
   ```
   - Приложение спит до 2 минут до волны
   - Вы можете оставить компьютер - приложение проснется автоматически
   - Обновления прогресса каждые 30 секунд

5. **Опрос перед волной** (за 2 минуты до волны)
   ```
   🔍 Начат опрос перед волной - проверка доступности страницы товара...
      Опрос: https://robertsspaceindustries.com/pledge/ships/...

      Статус 404 - Волна начнется через 1м 45с
      Статус 404 - Волна начнется через 1м 35с
   ```
   - Приложение проверяет каждую секунду, доступна ли страница продукта
   - Когда она изменяется с 404 на 200, это означает, что продажа началась!

6. **Продукт доступен!**
   ```
   ✅ Страница товара теперь доступна!
      (Страница появилась на 15с раньше запланированного времени)

   📄 Переход на страницу товара...
   🔍 Извлечение SKU со страницы товара...
   ```

7. **Попытка оформления заказа**
   ```
   🚀 Попытка оформления заказа...
      Таймаут наступит в: 19:05:00 МСК

   🔍 Проверка текущего состояния корзины...
   ✓ Корзина пуста, будет добавлен товар
   🛒 Добавление в корзину (API) с механизмом повторных попыток...
   🔄 Попытка 1 - Осталось времени: 4м59с
   🔄 Попытка 87 - Осталось времени: 4м57с
   ✅ Успешно добавлено в корзину после 87 попыток за 2.3с!

   💰 Применение кредита магазина...
   ➡️  Переход к этапу выставления счета/адресов...
   📍 Назначение адреса для выставления счета...
   🎯 Завершение заказа...

   ✅ Покупка успешна!
   👋 Выход из режима мультиволн (оформление успешно завершено)
   ```
   - Если успешно: **Приложение завершается корректно** - ВЫ ПОЛУЧИЛИ КОРАБЛЬ!
   - Если не удалось: Приложение продолжает пытаться еще 5 минут, затем переходит к следующей волне

8. **Если волна не удалась** (автоматически остается в состоянии ожидания до следующей волны)
   ```
   ❌ Волна 2: Оформление не удалось
   ➡️  Переход к волне 3...
      Следующая волна начнется через: 3ч 52м 18с
   💤 Ожидание до времени активации следующей волны
   ```
   - Приложение терпеливо ждет следующей волны
   - Вам не нужно перезапускать приложение
   - Процесс повторяется для всех оставшихся волн

9. **Если последняя волна не удалась**
   ```
   ❌ Волна 6: Оформление не удалось
      Это была последняя волна - волн больше не осталось

   ❌ Все волны завершены без успешного оформления
   ```
   - Приложение информирует вас, что все волны были обработаны
   - К сожалению, корабль распродан

### Расширенные опции

Вы можете настроить время волн с помощью флагов командной строки:

**Изменить время опроса перед волной** (по умолчанию: за 2 минуты)
```
specter.exe --pre-wave 3
```
Это начинает проверять страницу продукта за 3 минуты до каждой волны вместо 2.

**Изменить таймаут после волны** (по умолчанию: 5 минут после)
```
specter.exe --post-wave 10
```
Это продолжает пытаться 10 минут после каждой волны перед переходом к следующей.

**Комбинировать оба:**
```
specter.exe --pre-wave 3 --post-wave 10
```

**Режим отладки** (см. подробную техническую информацию):
```
specter.exe --debug
```

### Устранение неполадок

**"No sale windows configured"**
- Вы забыли добавить времена продаж в config.yaml
- Вернитесь к Настройке Шаг 1

**"All sale waves have already ended!"** (Все волны распродажи уже завершились!)
- Вы запустили приложение после окончания всех волн
- Последняя волна завершилась во время, показанное на экране
- Проверьте Spectrum/Discord RSI на наличие потенциальных дополнительных волн или будущих продаж

**"Product page never became available (timed out)"**
- Продажа корабля может быть отменена или отложена
- Проверьте сайт RSI или Discord для обновлений
- Ваши времена продаж могут быть неправильными (проверьте часовой пояс - должен быть UTC!)

**"All waves completed without successful checkout"**
- Корабль распродан во всех волнах до того, как вы смогли его получить
- К сожалению, это означает, что он недоступен
- Проверьте Spectrum/Discord на предмет потенциальных дополнительных волн

**"Browser is not responding"**
- Закройте все окна Chrome полностью
- На Windows: Откройте Диспетчер задач → Завершите все процессы Chrome
- На Mac: Command+Q чтобы выйти из Chrome, или Мониторинг системы → Принудительное завершение Chrome
- Попробуйте запустить Specter снова

**Windows Defender блокирует программу**
- Это ложное срабатывание (распространено с новыми программами)
- Нажмите "Подробнее" → "Выполнить в любом случае"
- Или добавьте исключение: Безопасность Windows → Защита от вирусов и угроз → Управление настройками → Добавить исключение → Выберите папку Specter

**Mac говорит "cannot be opened because it is from an unidentified developer"**
- Щелкните правой кнопкой `specter` → Открыть
- Нажмите "Открыть" в предупреждении безопасности
- Или: Системные настройки → Конфиденциальность и безопасность → Разрешить приложения из: App Store и идентифицированных разработчиков

**Все еще есть проблемы?**
- Убедитесь, что вы успешно выполнили Шаг 2 (Первоначальный вход)
- Убедитесь, что Chrome установлен и обновлен
- Попробуйте флаг `--debug` чтобы увидеть подробные сообщения об ошибках

### Советы для успеха

1. **Настройтесь РАНО** - Выполните Шаг 1 и 2 как минимум за 30 минут до продажи
2. **Тестируйте с --dry-run** - Всегда тестируйте перед настоящей продажей!
3. **Проверьте свой кредит магазина** - Абсолютно убедитесь, что у вас достаточно
4. **Используйте стандартные времена волн CIG** - Дважды проверьте времена на Spectrum/Discord
5. **Не трогайте компьютер во время волн** - Позвольте приложению работать автоматически
6. **Держите окно браузера видимым** - Не сворачивайте его (может вызвать проблемы на некоторых системах)
7. **Хорошее интернет-соединение** - Используйте проводной Ethernet, если возможно

### Как это работает (Технические детали)

Для тех, кто интересуется технической реализацией:

1. **Синхронизация времени**: Приложение использует HTTP HEAD запросы к серверу времени Amazon для расчета точного смещения времени (инфраструктура CIG работает на AWS для точного таймирования)
2. **Умное определение волны**: При запуске сравнивает текущее время со временем окончания всех волн (время_волны + таймаут_после_волны), чтобы определить, с какой волны начать
3. **Пропуск прошедших волн**: Автоматически пропускает волны, которые уже закончились, отображает список пропущенных волн пользователю
4. **Опрос перед волной**: Начиная за 2 минуты до каждой волны, отправляет HTTP HEAD запросы каждую секунду, проверяя статус 200 (продукт доступен)
5. **Извлечение SKU**: Как только страница доступна, использует JavaScript-оценку браузера для извлечения SKU из нескольких источников (данные Next.js, теги скриптов, свойства компонентов)
6. **Оформление заказа через API**: Полностью обходит UI браузера, отправляет прямые GraphQL мутации к API магазина RSI
7. **Умные повторные попытки**: Реализует экспоненциальную задержку для ограничений скорости, специфические задержки для различных типов ошибок (4226, 4227, нет на складе и т.д.)
8. **Проверка корзины**: Определяет, есть ли в корзине уже правильный товар с примененными кредитами, пропускает избыточные шаги
9. **Кэширование адреса**: Предварительно получает и кэширует адрес для выставления счета, чтобы устранить задержки поиска во время оформления заказа
10. **reCAPTCHA v3**: Генерирует свежие Enterprise токены для каждой попытки добавления в корзину
11. **Мультиволновая машина состояний**: Автоматически переходит между волнами по таймауту, остается в состоянии ожидания между волнами, корректно завершается при успехе или когда все волны завершены
