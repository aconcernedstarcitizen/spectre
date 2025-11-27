# Specter - RSI Store Automated Checkout

**Lightning-fast automated checkout for limited Star Citizen ship sales**

[English](#english) | [Русский](#русский)

---

## English

### What is This?

Specter is a tool that automatically buys limited-edition ships from the Star Citizen store (robertsspaceindustries.com) at lightning speed using **store credit only**. When ships sell out in seconds, this gives you the best chance to complete your purchase.

**⚠️ IMPORTANT LIMITATIONS:**
- ✅ **Works ONLY with store credit** - Cannot process credit card or PayPal payments
- ✅ **Single ship purchases only** - Designed for buying one ship at a time
- ❌ Does NOT work for cash/credit card purchases
- ❌ Does NOT work for game packages or multi-item purchases

**Key Features:**
- ⚡ **Ultra-fast checkout** - Completes purchase in under 1 second once item is in cart!
- 🔄 **Aggressive retry system** - Attempts to add items 50-200 times per second with 5-20ms delays
- ⏰ **Timed Sale Mode** - Automatically starts trying 10 minutes before the sale and continues for 20 minutes after
- 💳 **Automatic store credit application** - No manual steps needed
- 🛡️ **Cart validation safeguards** - Prevents accidentally buying multiple items or wrong ships
- 🤖 **Smart rate limit handling** - Automatically adjusts if the server is busy
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

#### Step 1: First-Time Login

1. **Open the folder** where you saved Specter

2. **For Windows:**
   - Double-click `specter.exe`
   - If Windows says "Windows protected your PC", click "More info" then "Run anyway"

   **For Mac:**
   - Open Terminal
   - Type `cd ` (with space)
   - Drag the Specter folder into Terminal and press Enter
   - Type: `./specter` and press Enter

3. **A Chrome window will open** - this is normal

4. **Log in to your RSI account** in this Chrome window

5. **The program will say "No item URL specified"** - this is expected!
   - Just wait in the browser and press ENTER when ready

6. Your login is now saved! Close everything.

#### Step 2: Configure the Ship URL

1. **Find config.yaml** in the Specter folder (included in the download)

2. **Open it with Notepad (Windows) or TextEdit (Mac)**

3. **Find the line that says:** `item_url: ""`

4. **Put the ship URL between the quotes.** For example:
   ```yaml
   item_url: "https://robertsspaceindustries.com/pledge/ships/anvil-carrack/Carrack"
   ```

5. **Save the file** (File → Save)

#### Step 3: Test It!

**Do a test run with a cheap ship you don't mind buying:**

**Windows:**
- Open Command Prompt (search for "cmd")
- Type: `cd C:\Specter` (or wherever you saved it)
- Type: `specter.exe --dry-run`
- Press Enter

**Mac:**
- Open Terminal
- Type: `cd /Users/YourName/Specter` (or wherever you saved it)
- Type: `./specter --dry-run`
- Press Enter

The program will go through the whole process but stop before actually buying. This confirms everything works!

### How to Use - Two Modes

Specter has **two modes**: Normal Mode (for immediate purchases) and Timed Sale Mode (for scheduled sales).

---

#### Normal Mode - For Immediate Purchases

**Use this when:** You want to buy a ship that's available right now, or manually control when to start.

**Windows:**
```
cd C:\Specter
specter.exe --url "https://robertsspaceindustries.com/pledge/ships/..."
```

**Mac:**
```
cd /Users/YourName/Specter
./specter --url "https://robertsspaceindustries.com/pledge/ships/..."
```

**What happens:**
1. Chrome opens - log in if needed
2. **Press ENTER** when you're ready to start
   (The app will retry until the item becomes available)
3. Program tries to add to cart with ultra-fast retries (5-20ms between attempts)
4. Once successful, completes checkout in under 1 second
5. Done! Your order is placed

**What You'll See:**
```
╔═══════════════════════════════════════════════════════════╗
║           FAST CHECKOUT - API MODE                        ║
║           (Browser-Free Lightning Speed)                  ║
╚═══════════════════════════════════════════════════════════╝

🔍 Checking current cart state...
✓ Cart is empty, will add item
🛒 Adding to cart (API) with retry mechanism...
⏱️  Will retry for up to 300 seconds
🔄 Attempt 1 - Time remaining: 4m59s
🔄 Attempt 50 - Time remaining: 4m58s
✅ Successfully added to cart after 87 attempts in 2.3s!

✓ Item added successfully (validation skipped for speed)
💰 Applying $20.00 store credit (API)...
✓ Store credit applied: [success response]

➡️  Moving to billing/addresses step...
📋 Fetching billing address...
✓ Found billing address: John Doe, Los Angeles (ID: abc123)
📍 Assigning billing address (ID: abc123)...
✓ Billing address assigned: [success response]

🎯 Completing order (validating cart)...
✓ Order validated! Order slug: order-12345
✓ Order has been created!
✓ ORDER COMPLETED!

⚡ Total checkout time: 847ms
🎯 Target: <1 second | Actual: 847ms
🏆 ACHIEVED SUB-SECOND CHECKOUT!
```

---

#### Timed Sale Mode - For Scheduled Sales

**Use this when:** You know the exact time a limited ship goes on sale (like Kraken, Idris, etc.)

**What is Timed Sale Mode?**
- You tell Specter when the sale starts (exact date and time)
- It **automatically starts trying 10 minutes before** the sale
- **Hammers the server** with 50-200 attempts per second
- **Continues for 20 minutes after** the sale starts
- You don't need to press ENTER or do anything - it's fully automatic!

**How to Use:**

1. **Find out the sale time** - For example: "Kraken sale on January 15, 2025 at 6:00 PM EST"

2. **Convert to UTC time** (use worldtimebuddy.com or Google "EST to UTC")
   - Example: 6:00 PM EST = 11:00 PM UTC = 23:00

3. **Run with the sale time:**

**Windows:**
```
cd C:\Specter
specter.exe --url "https://robertsspaceindustries.com/pledge/ships/..." --sale-time "2025-01-15T23:00:00Z"
```

**Mac:**
```
cd /Users/YourName/Specter
./specter --url "https://robertsspaceindustries.com/pledge/ships/..." --sale-time "2025-01-15T23:00:00Z"
```

**Time Format:** `YYYY-MM-DDTHH:MM:SSZ` (always end with Z for UTC time)
- January 15, 2025 at 11:00 PM UTC = `2025-01-15T23:00:00Z`
- December 25, 2024 at 6:30 PM UTC = `2024-12-25T18:30:00Z`

**Customize the timing (optional):**
```
specter.exe --url "..." --sale-time "2025-01-15T23:00:00Z" --start-before 15 --continue-after 30
```
- `--start-before 15` = Start trying 15 minutes before sale (default: 10)
- `--continue-after 30` = Keep trying 30 minutes after sale starts (default: 20)

**What happens:**
1. Chrome opens - log in if needed
2. Press ENTER to confirm you're logged in
3. Program waits until 10 minutes before sale
4. **Automatically starts hammering add-to-cart** with ultra-fast retries
5. Once item is added, completes checkout in under 1 second
6. Done!

**What You'll See:**
```
╔═══════════════════════════════════════════════════════════╗
║           TIMED SALE MODE - AGGRESSIVE RETRY              ║
╚═══════════════════════════════════════════════════════════╝

⏰ Sale starts at: Wed, 15 Jan 2025 23:00:00 UTC
🚀 Will start retrying at: Wed, 15 Jan 2025 22:50:00 UTC (10 min before)
⏱️  Will stop retrying at: Wed, 15 Jan 2025 23:20:00 UTC (20 min after)

🔧 Running pre-flight checks...
🔐 Extracting session from browser...
✓ Extracted 12 cookies from browser
✓ Extracted CSRF token: abc123...
🔍 Extracting SKU from current page...
✓ Extracted SKU slug from page: anvil-carrack
✓ Found SKU ID: 12345
✅ All pre-flight checks passed!

💡 Tip: Everything is ready! You can take a break until the sale starts.

⏳ Waiting 8m 45s until retry window starts...
✓ Retry window started!

═══════════════════════════════════════════════════════════
           PHASE 1: ADD TO CART (AGGRESSIVE RETRY)
═══════════════════════════════════════════════════════════
🔄 Attempt 1 - Time remaining: 30m0s
🔄 Attempt 50 - Time remaining: 29m59s
🔄 Attempt 100 - Time remaining: 29m59s
✅ Successfully added to cart after 247 attempts in 4.8s!

═══════════════════════════════════════════════════════════
           PHASE 2: CHECKOUT (AGGRESSIVE RETRY)
═══════════════════════════════════════════════════════════
➡️  Moving to billing/addresses step...
💰 Applying $350.00 store credit (API)...
✓ Store credit applied: [success response]
✅ Validating and completing order...
🔄 Validation Attempt 1 - Time remaining in sale window: 29m54s
✓ Order validated! Order slug: order-12345
✓ Order has been created!
🎉 Success after 1 attempt(s) in 523ms
✓ ORDER COMPLETED!

⚡ Total time from first attempt to completion: 5.2s
```

---

### Cart Validation Safety Features

Specter includes **automatic cart validation** to protect you from accidentally buying wrong items or multiple ships:

**What it checks:**
- ✓ Only **1 item** in cart (no accidentally buying multiple different ships)
- ✓ Item **quantity is 1** (not buying 5x of the same ship)
- ✓ **Correct SKU** matches your target URL
- ✓ **Cart total matches** the expected single item price

**When it validates:**
1. **Before adding to cart** - Checks if cart already has items
2. **After adding to cart** - Confirms correct item was added
3. **Before applying store credit** - Final check before purchase

**If cart is empty:**
```
🔍 Checking current cart state...
✓ Cart is empty, will add item
```
→ Proceeds normally, adds item to cart

**If cart already has the correct item:**
```
🔍 Checking current cart state...
✓ Cart already contains target item: Aurora ES - 10 Year ($20.00)
  Skipping add-to-cart step (would create duplicate)
```
→ Skips adding, proceeds to checkout with existing item

**If cart has issues (wrong items, multiple items, etc.):**
```
╔═══════════════════════════════════════════════════════════╗
║                    ⚠️  CART WARNING                       ║
╚═══════════════════════════════════════════════════════════╝

Your cart contains 5 × Aurora ES - 10 Year:

→ 1. Aurora ES - 10 Year (Quantity: 5)
   Price: $20.00 × 5 = $100.00
   (This is your target item)
   ⚠️  WARNING: Buying 5 copies of this ship!

Cart Total: $100.00

⚠️  You are buying 5 copies of the SAME ship!
   This will purchase 5 × Aurora ES - 10 Year for $100.00 total.

   NOTE: RSI limits purchases to max 5 of any item per order.

Options:
  1. Press ENTER to continue with the CURRENT cart contents
  2. Press ESC to cancel and manually edit your cart
```
→ You can choose to proceed with existing cart or cancel to fix it

This prevents accidentally purchasing the wrong items during high-pressure limited sales!

---

### Settings You Can Change

Open `config.yaml` to customize:

#### Basic Settings:
```yaml
item_url: ""  # Ship URL - can also use --url flag
auto_apply_credit: true  # Automatically use store credit
dry_run: false  # Set to true for test mode (doesn't actually buy)
```

#### Retry Settings:
```yaml
retry_duration_seconds: 300  # How long to keep trying (5 minutes default)
retry_delay_min_ms: 5        # Minimum delay between attempts (5ms - ultra fast!)
retry_delay_max_ms: 20       # Maximum delay between attempts (20ms)
```

#### Timed Sale Settings:
```yaml
enable_sale_timing: false  # Set to true to use timed mode via config
sale_start_time: ""        # e.g., "2025-01-15T23:00:00Z"
start_before_sale_minutes: 10   # Start trying X minutes before
continue_after_sale_minutes: 20  # Keep trying X minutes after
```

**Note:** Using command-line flags (`--sale-time`, `--start-before`, etc.) will override these config settings.

### Common Questions

**Q: Will this get me banned?**
A: Using automation tools may violate RSI's Terms of Service. Use at your own risk. This tool is designed to be respectful (it detects rate limits and backs off), but there's always a risk.

**Q: How fast is it?**
A: The checkout completes in **under 1 second** once the ship is in your cart. The retry system attempts **50-200 times per second** with 5-20ms delays, making it extremely competitive for limited sales.

**Q: What's the difference between Normal and Timed Mode?**
A:
- **Normal Mode:** You control when to start by pressing ENTER. Good for manual timing or items already available.
- **Timed Mode:** Fully automatic. You set the sale time and it handles everything - starts early, retries aggressively, completes purchase. No button pressing needed!

**Q: Do I need programming experience?**
A: No! Just follow the instructions above. If you can open files and type commands, you can use this.

**Q: What if it doesn't work?**
A: Make sure you:
- Logged in successfully (Step 1)
- Put the correct ship URL (check it in your browser first)
- **Have enough store credit in your account to cover the FULL ship price**
- The ship is a standalone ship (not a game package)
- Have a fast internet connection
- Used the correct time format for timed mode (ending with Z)

**Q: Can I use a credit card or PayPal instead of store credit?**
A: **No.** Specter ONLY works with store credit purchases. You must have sufficient store credit in your RSI account before running the tool. If you don't have enough store credit, the purchase will fail.

**Q: Can I use this for multiple ships?**
A: Yes! Create multiple config files (like `carrack.yaml`, `idris.yaml`) with different URLs, then run: `specter.exe --config carrack.yaml`

**Q: The program says "rate limited" - what does that mean?**
A: The server is busy and asked us to slow down. The program automatically waits 50-150ms (instead of 5-20ms) before trying again. This is normal during busy sales!

**Q: What time zone should I use for timed mode?**
A: Always use **UTC time** and end with `Z`. Convert your local time to UTC first using worldtimebuddy.com or Google.

### Troubleshooting

**"No item URL specified"**
- You forgot to put the ship URL in config.yaml OR forgot to use --url flag
- The config.yaml file is included in the download - make sure you extracted the full ZIP

**"Failed to launch browser"**
- Make sure Chrome is installed (strongly recommended)
- Try deleting the `.specter` folder in your home directory and login again

**"macOS Security Warning" or "iTerm has prevented an app from modifying files" (Mac)**
This is a macOS security feature that prevents terminal apps from creating files in certain locations:

**Best fix: Grant Terminal Full Disk Access**
1. Open **System Settings** (or System Preferences on older macOS)
2. Go to **Privacy & Security** → **Full Disk Access**
3. Click the lock icon and enter your password
4. Click the **+** button and add your terminal app:
   - If using iTerm: Select **iTerm.app** from Applications
   - If using Terminal: Select **Terminal.app** from Applications/Utilities
5. Enable the checkbox next to your terminal app
6. **Restart your terminal app** (quit completely and reopen)
7. Try running Specter again

**Alternative: Use Terminal.app instead of iTerm**
- The built-in Terminal.app often has necessary permissions by default
- Open Terminal.app from Applications/Utilities
- Navigate to Specter folder and run `./specter`

**"Chrome is already running" or "ProcessSingleton" / "SingletonLock" error**
This happens when Chrome is already open and using the same profile:

1. **Close ALL Chrome windows completely**
   - Windows: Check Task Manager → End all Chrome.exe processes
   - Mac: Check Activity Monitor → Quit all Chrome processes
   - Or on Mac Terminal: `killall 'Google Chrome'`

2. **Try running Specter again**
   - Specter will launch Chrome with its own isolated profile
   - Your existing Chrome data won't be affected

**"Browser download failed due to file permissions" (Windows)**
This happens when Specter tries to download a temporary browser but encounters permission issues:

**Best fix: Install Google Chrome**
- Download from: https://www.google.com/chrome
- Specter will automatically use your Chrome installation
- No downloads or permission issues

**Alternative fix: Clear browser cache**
1. Close ALL Chrome/Chromium processes (check Task Manager)
2. Press `Win+R`, type: `%APPDATA%\rod` and press Enter
3. Delete the `browser` folder
4. Add antivirus exclusion for `%APPDATA%\rod` folder (see antivirus section below)
5. Try running Specter again

**"Session expired"**
- Your login expired
- Run the program again and it will open Chrome for you to login

**"Insufficient store credit available" error**
- You don't have enough store credit in your RSI account to cover the ship price
- Check your store credit balance at robertsspaceindustries.com
- Add more store credit to your account or choose a less expensive ship
- The app ONLY works with store credit - you cannot use credit cards or PayPal

**"Invalid sale start time format"**
- Make sure you use the correct format: `YYYY-MM-DDTHH:MM:SSZ`
- Always end with `Z` for UTC time
- Example: `2025-01-15T23:00:00Z`

**Program exits immediately on Windows**
- You might need to allow it through Windows Defender
- Right-click specter.exe → Properties → Unblock → Apply

**"Sale window has already passed"**
- The time you specified has already happened
- Check your time conversion (make sure you used UTC, not local time)
- Make sure the date is correct

**Antivirus is blocking/flagging the program (Kaspersky, Windows Defender, etc.)**

This is a **false positive** - extremely common for browser automation tools. Here's why:

**Why does this happen?**
- The program launches browsers, controls them remotely, and makes network requests
- Antivirus software can't tell the difference between legitimate automation and malicious automation
- The program is NOT code-signed (requires expensive certificate ~$300/year)
- Common detections: "PDM:Trojan.Bazon.a", "Trojan:Win32/Wacatac", or similar

**The program is safe:**
- ✅ 100% open source - you can review all code on GitHub
- ✅ No data collection - everything runs locally
- ✅ No network access except to RSI's official website
- ✅ Builds are automated via GitHub Actions (visible in repository)

**How to fix:**
1. **Kaspersky:** Settings → Threats and Exclusions → Manage Exclusions → Add → Browse to `specter.exe`
2. **Windows Defender:** Windows Security → Virus & threat protection → Manage settings → Exclusions → Add exclusion → File → Select `specter.exe`
3. **Other antivirus:** Look for "Exclusions", "Whitelist", or "Trusted Applications" in settings

**Still concerned?**
- Review the source code yourself on GitHub
- Build from source instead of using pre-built binaries
- Check the file hash against the one published on the releases page
- Run in a virtual machine if you want extra isolation

### Support

Need help? Check the issues page on GitHub or ask in the Star Citizen community.

### Legal Disclaimer

⚠️ **Use at your own risk.** This tool may violate RSI's Terms of Service. The authors are not responsible for any consequences including account suspension. This software is provided "as is" without warranty.

This tool does not collect any data. Everything runs locally on your computer. Your RSI password is handled only by Chrome, never by this program.

---

## Русский

### Что это?

Specter - это инструмент, который автоматически покупает лимитированные корабли из магазина Star Citizen (robertsspaceindustries.com) с молниеносной скоростью используя **только store credit**. Когда корабли распродаются за секунды, это дает вам лучший шанс завершить покупку.

**⚠️ ВАЖНЫЕ ОГРАНИЧЕНИЯ:**
- ✅ **Работает ТОЛЬКО со store credit** - Не может обрабатывать оплату картой или PayPal
- ✅ **Только покупки одного корабля** - Разработан для покупки одного корабля за раз
- ❌ НЕ работает для покупок за наличные/карту
- ❌ НЕ работает для игровых пакетов или покупок нескольких предметов

**Основные возможности:**
- ⚡ **Сверхбыстрое оформление** - Завершает покупку менее чем за 1 секунду после добавления в корзину!
- 🔄 **Агрессивная система повторов** - 50-200 попыток в секунду с задержками 5-20мс
- ⏰ **Режим по расписанию** - Автоматически начинает попытки за 10 минут до продажи и продолжает 20 минут после
- 💳 **Автоматическое применение store credit** - Без ручных действий
- 🛡️ **Защита проверки корзины** - Предотвращает случайную покупку нескольких предметов или неправильных кораблей
- 🤖 **Умная обработка ограничений** - Автоматически подстраивается если сервер занят
- 🎯 **Оптимизирован для скорости** - Каждая миллисекунда важна при конкуренции за лимитированные корабли
- 🌍 **Многоязычная поддержка** - Автоматически определяет язык системы (поддерживается английский, русский)

### Требования

**Что вам нужно:**
- Компьютер (Windows 10/11 или Mac)
- Установленный браузер Google Chrome (настоятельно рекомендуется - избегает проблем с загрузкой)
- **Аккаунт Star Citizen с ДОСТАТОЧНЫМ количеством store credit** - Приложение работает ТОЛЬКО с оплатой store credit
- Корабль должен быть доступен для покупки как отдельный standalone предмет (не пакет)
- Базовые навыки работы с компьютером (открытие файлов, запуск программ)

**⚠️ КРИТИЧЕСКИ ВАЖНО:** У вас должно быть достаточно store credit в вашем аккаунте RSI чтобы покрыть полную стоимость корабля. Приложение не может использовать кредитные карты, PayPal или любой другой метод оплаты.

**Примечание:** Specter автоматически использует ваш установленный Chrome браузер если доступен. Если Chrome не установлен, он загрузит временный браузер (может потребоваться добавление исключений в антивирус на Windows).

### Установка

#### Для Windows:

1. **Скачайте Specter:**
   - Перейдите по ссылке: **https://github.com/aconcernedstarcitizen/spectre/releases**
   - Скачайте последний файл `specter-windows-amd64.zip` (смотрите в разделе "Assets")
   - **Щелкните правой кнопкой мыши на ZIP файл** и выберите "Извлечь все..."
   - Извлеките в папку (например `C:\Specter`)
   - Извлеченная папка будет содержать:
     - `specter.exe` - Программа
     - `config.yaml` - Файл конфигурации
     - `lang/` - Языковые файлы (автоматически определяет язык системы)

2. **Убедитесь что Chrome установлен:**
   - Если у вас нет Chrome, скачайте его с google.com/chrome

#### Для Mac:

1. **Скачайте Specter:**
   - Перейдите по ссылке: **https://github.com/aconcernedstarcitizen/spectre/releases**
   - Скачайте последний ZIP файл для Mac:
     - `specter-macos-arm64.zip` если у вас Apple Silicon (M1/M2/M3/M4)
     - `specter-macos-amd64.zip` если у вас Intel Mac
   - **Дважды кликните на ZIP файл** чтобы извлечь его
   - Переместите извлеченную папку в место типа `/Users/ВашеИмя/Specter`
   - Извлеченная папка будет содержать:
     - `specter` - Программа
     - `config.yaml` - Файл конфигурации
     - `lang/` - Языковые файлы (автоматически определяет язык системы)

2. **Сделайте его запускаемым:**
   - Откройте Terminal (найдите "Terminal" через Spotlight)
   - Введите: `cd ` (с пробелом в конце)
   - Перетащите папку Specter в Terminal и нажмите Enter
   - Введите: `chmod +x specter` и нажмите Enter

3. **Убедитесь что Chrome установлен:**
   - Если у вас нет Chrome, скачайте его с google.com/chrome

### Настройка (Сделайте это перед продажей!)

**ВАЖНО: Выполните эти шаги минимум за 30 минут до продажи корабля!**

#### Шаг 1: Первый вход

1. **Откройте папку**, где вы сохранили Specter

2. **Для Windows:**
   - Дважды кликните на `specter.exe`
   - Если Windows говорит "Windows защитил ваш ПК", нажмите "Подробнее" затем "Выполнить в любом случае"

   **Для Mac:**
   - Откройте Terminal
   - Введите `cd ` (с пробелом)
   - Перетащите папку Specter в Terminal и нажмите Enter
   - Введите: `./specter` и нажмите Enter

3. **Откроется окно Chrome** - это нормально

4. **Войдите в ваш аккаунт RSI** в этом окне Chrome

5. **Программа скажет "No item URL specified"** - это ожидаемо!
   - Просто подождите в браузере и нажмите ENTER когда готовы

6. Ваш вход теперь сохранен! Закройте все.

#### Шаг 2: Настройте URL корабля

1. **Найдите config.yaml** в папке Specter (включен в загрузку)

2. **Откройте его Блокнотом (Windows) или TextEdit (Mac)**

3. **Найдите строку:** `item_url: ""`

4. **Вставьте URL корабля между кавычек.** Например:
   ```yaml
   item_url: "https://robertsspaceindustries.com/pledge/ships/anvil-carrack/Carrack"
   ```

5. **Сохраните файл** (Файл → Сохранить)

#### Шаг 3: Протестируйте!

**Сделайте тестовый запуск с дешевым кораблем, который не жалко купить:**

**Windows:**
- Откройте Командную строку (найдите "cmd")
- Введите: `cd C:\Specter` (или где вы его сохранили)
- Введите: `specter.exe --dry-run`
- Нажмите Enter

**Mac:**
- Откройте Terminal
- Введите: `cd /Users/ВашеИмя/Specter` (или где вы его сохранили)
- Введите: `./specter --dry-run`
- Нажмите Enter

Программа пройдет весь процесс но остановится перед реальной покупкой. Это подтверждает что все работает!

### Как использовать - Два режима

У Specter есть **два режима**: Обычный режим (для немедленных покупок) и Режим по расписанию (для запланированных продаж).

---

#### Обычный режим - Для немедленных покупок

**Используйте когда:** Хотите купить корабль который доступен прямо сейчас, или контролировать запуск вручную.

**Windows:**
```
cd C:\Specter
specter.exe --url "https://robertsspaceindustries.com/pledge/ships/..."
```

**Mac:**
```
cd /Users/ВашеИмя/Specter
./specter --url "https://robertsspaceindustries.com/pledge/ships/..."
```

**Что происходит:**
1. Открывается Chrome - войдите если нужно
2. **Нажмите ENTER** когда готовы начать
   (Приложение будет повторять попытки пока товар не станет доступен)
3. Программа пытается добавить в корзину со сверхбыстрыми повторами (5-20мс между попытками)
4. После успеха завершает оформление менее чем за 1 секунду
5. Готово! Ваш заказ размещен

**Что вы увидите (на русском языке, если система настроена на русский):**
```
╔═══════════════════════════════════════════════════════════╗
║           FAST CHECKOUT - API MODE                        ║
║           (Browser-Free Lightning Speed)                  ║
╚═══════════════════════════════════════════════════════════╝

🔍 Проверка текущего состояния корзины...
✓ Корзина пуста, будет добавлен товар
🛒 Добавление в корзину (API) с механизмом повторов...
⏱️  Будет повторяться до 300 секунд
🔄 Попытка 1 - Осталось времени: 4m59s
🔄 Попытка 50 - Осталось времени: 4m58s
✅ Успешно добавлено в корзину после 87 попыток за 2.3s!

✓ Товар успешно добавлен (проверка пропущена для скорости)
💰 Применение $20.00 store credit (API)...
✓ Store credit применён: [успешный ответ]

➡️  Переход к шагу оплаты/адресов...
📋 Получение платёжного адреса...
✓ Найден платёжный адрес: Иван Иванов, Москва (ID: abc123)
📍 Назначение платёжного адреса (ID: abc123)...
✓ Платёжный адрес назначен: [успешный ответ]

🎯 Завершение заказа (проверка корзины)...
✓ Заказ подтверждён! Номер заказа: order-12345
✓ Заказ создан!
✓ ЗАКАЗ ЗАВЕРШЕН!

⚡ Общее время оформления: 847ms
🎯 Цель: <1 секунда | Факт: 847ms
🏆 ДОСТИГНУТО ОФОРМЛЕНИЕ МЕНЕЕ СЕКУНДЫ!
```

---

#### Режим по расписанию - Для запланированных продаж

**Используйте когда:** Вы знаете точное время когда лимитированный корабль поступит в продажу (как Kraken, Idris, и т.д.)

**Что такое режим по расписанию?**
- Вы говорите Specter когда начинается продажа (точная дата и время)
- Он **автоматически начинает попытки за 10 минут до** продажи
- **Бомбардирует сервер** 50-200 попытками в секунду
- **Продолжает 20 минут после** начала продажи
- Вам не нужно нажимать ENTER или что-то делать - все полностью автоматически!

**Как использовать:**

1. **Узнайте время продажи** - Например: "Продажа Kraken 15 января 2025 в 6:00 PM EST"

2. **Конвертируйте в UTC время** (используйте worldtimebuddy.com или Google "EST to UTC")
   - Пример: 6:00 PM EST = 11:00 PM UTC = 23:00

3. **Запустите со временем продажи:**

**Windows:**
```
cd C:\Specter
specter.exe --url "https://robertsspaceindustries.com/pledge/ships/..." --sale-time "2025-01-15T23:00:00Z"
```

**Mac:**
```
cd /Users/ВашеИмя/Specter
./specter --url "https://robertsspaceindustries.com/pledge/ships/..." --sale-time "2025-01-15T23:00:00Z"
```

**Формат времени:** `YYYY-MM-DDTHH:MM:SSZ` (всегда заканчивайте на Z для UTC времени)
- 15 января 2025 в 11:00 PM UTC = `2025-01-15T23:00:00Z`
- 25 декабря 2024 в 6:30 PM UTC = `2024-12-25T18:30:00Z`

**Настройка времени (опционально):**
```
specter.exe --url "..." --sale-time "2025-01-15T23:00:00Z" --start-before 15 --continue-after 30
```
- `--start-before 15` = Начать попытки за 15 минут до продажи (по умолчанию: 10)
- `--continue-after 30` = Продолжать попытки 30 минут после начала (по умолчанию: 20)

**Что происходит:**
1. Открывается Chrome - войдите если нужно
2. Нажмите ENTER для подтверждения что вы вошли
3. Программа ждет до 10 минут перед продажей
4. **Автоматически начинает бомбардировку add-to-cart** со сверхбыстрыми повторами
5. После добавления товара завершает оформление менее чем за 1 секунду
6. Готово!

**Что вы увидите:**
```
╔═══════════════════════════════════════════════════════════╗
║           TIMED SALE MODE - AGGRESSIVE RETRY              ║
╚═══════════════════════════════════════════════════════════╝

⏰ Sale starts at: Wed, 15 Jan 2025 23:00:00 UTC
🚀 Will start retrying at: Wed, 15 Jan 2025 22:50:00 UTC (10 min before)
⏱️  Will stop retrying at: Wed, 15 Jan 2025 23:20:00 UTC (20 min after)

⏳ Waiting 8m 45s until retry window starts...
✓ Retry window started!

═══════════════════════════════════════════════════════════
           PHASE 1: ADD TO CART (AGGRESSIVE RETRY)
═══════════════════════════════════════════════════════════
🔄 Attempt 1 - Time remaining: 30m0s
🔄 Attempt 50 - Time remaining: 29m59s
🔄 Attempt 100 - Time remaining: 29m59s
✅ Successfully added to cart after 247 attempts in 4.8s!

═══════════════════════════════════════════════════════════
           PHASE 2: CHECKOUT (AGGRESSIVE RETRY)
═══════════════════════════════════════════════════════════
➡️  Moving to billing step...
💰 Applying store credit...
✓ ORDER COMPLETED!

⚡ Total time from first attempt to completion: 5.2s
```

---

### Функции защиты проверки корзины

Specter включает **автоматическую проверку корзины** для защиты от случайной покупки неправильных товаров или нескольких кораблей:

**Что проверяется:**
- ✓ Только **1 товар** в корзине (никакой случайной покупки нескольких разных кораблей)
- ✓ **Количество товара 1** (не покупка 5x одного корабля)
- ✓ **Правильный SKU** соответствует вашему целевому URL
- ✓ **Итог корзины совпадает** с ожидаемой ценой одного товара

**Когда проверяется:**
1. **Перед добавлением в корзину** - Проверка есть ли уже товары в корзине
2. **После добавления в корзину** - Подтверждение что добавлен правильный товар
3. **Перед применением store credit** - Финальная проверка перед покупкой

**Если корзина пуста:**
```
🔍 Проверка текущего состояния корзины...
✓ Корзина пуста, будет добавлен товар
```
→ Продолжает нормально, добавляет товар в корзину

**Если в корзине уже правильный товар:**
```
🔍 Проверка текущего состояния корзины...
✓ Корзина уже содержит целевой товар: Aurora ES - 10 Year ($20.00)
  Пропуск шага добавления в корзину (создаст дубликат)
```
→ Пропускает добавление, переходит к оформлению с существующим товаром

**Если в корзине проблемы (неправильные товары, несколько товаров, и т.д.):**
```
╔═══════════════════════════════════════════════════════════╗
║                 ⚠️  ПРЕДУПРЕЖДЕНИЕ КОРЗИНЫ                ║
╚═══════════════════════════════════════════════════════════╝

Ваша корзина содержит 5 × Aurora ES - 10 Year:

→ 1. Aurora ES - 10 Year (Количество: 5)
   Цена: $20.00 × 5 = $100.00
   (Это ваш целевой товар)
   ⚠️  ВНИМАНИЕ: Покупка 5 копий этого корабля!

Итого корзины: $100.00

⚠️  Вы покупаете 5 копий ОДНОГО корабля!
   Это купит 5 × Aurora ES - 10 Year за $100.00 всего.

   ПРИМЕЧАНИЕ: RSI ограничивает покупки максимум 5 штук любого товара за заказ.

Опции:
  1. Нажмите ENTER чтобы продолжить с ТЕКУЩИМ содержимым корзины
  2. Нажмите ESC чтобы отменить и вручную отредактировать корзину
```
→ Вы можете выбрать продолжить с существующей корзиной или отменить чтобы исправить

Это предотвращает случайную покупку неправильных товаров во время стрессовых лимитированных продаж!

---

### Настройки которые можно изменить

Откройте `config.yaml` для настройки:

#### Базовые настройки:
```yaml
item_url: ""  # URL корабля - можно также использовать флаг --url
auto_apply_credit: true  # Автоматически использовать store credit
dry_run: false  # Установите true для тестового режима (не покупает на самом деле)
```

#### Настройки повторов:
```yaml
retry_duration_seconds: 300  # Как долго пытаться (5 минут по умолчанию)
retry_delay_min_ms: 5        # Минимальная задержка между попытками (5мс - сверхбыстро!)
retry_delay_max_ms: 20       # Максимальная задержка между попытками (20мс)
```

#### Настройки режима по расписанию:
```yaml
enable_sale_timing: false  # Установите true для использования режима через config
sale_start_time: ""        # например, "2025-01-15T23:00:00Z"
start_before_sale_minutes: 10   # Начать попытки за X минут до
continue_after_sale_minutes: 20  # Продолжать попытки X минут после
```

**Примечание:** Использование флагов командной строки (`--sale-time`, `--start-before`, и т.д.) переопределит эти настройки config.

### Частые вопросы

**В: Меня забанят за это?**
О: Использование инструментов автоматизации может нарушать Условия использования RSI. Используйте на свой риск. Этот инструмент разработан быть уважительным (определяет ограничения и замедляется), но риск всегда есть.

**В: Насколько это быстро?**
О: Оформление завершается **менее чем за 1 секунду** после того как корабль в корзине. Система повторов делает **50-200 попыток в секунду** с задержками 5-20мс, что делает его чрезвычайно конкурентоспособным для лимитированных продаж.

**В: В чем разница между Обычным и Режимом по расписанию?**
О:
- **Обычный режим:** Вы контролируете когда начать нажатием ENTER. Подходит для ручного контроля времени или уже доступных товаров.
- **Режим по расписанию:** Полностью автоматический. Вы устанавливаете время продажи и он делает все - начинает рано, агрессивно повторяет, завершает покупку. Нажимать кнопки не нужно!

**В: Нужен ли опыт программирования?**
О: Нет! Просто следуйте инструкциям выше. Если вы можете открывать файлы и вводить команды, вы можете это использовать.

**В: Что если не работает?**
О: Убедитесь что вы:
- Успешно вошли (Шаг 1)
- Вставили правильный URL корабля (проверьте его в браузере сначала)
- **Имеете достаточно store credit в аккаунте чтобы покрыть ПОЛНУЮ стоимость корабля**
- Корабль это отдельный standalone корабль (не игровой пакет)
- Имеете быстрое интернет-соединение
- Использовали правильный формат времени для режима по расписанию (заканчивающийся на Z)

**В: Могу ли я использовать кредитную карту или PayPal вместо store credit?**
О: **Нет.** Specter работает ТОЛЬКО с покупками за store credit. У вас должно быть достаточно store credit в вашем аккаунте RSI перед запуском инструмента. Если у вас недостаточно store credit, покупка не удастся.

**В: Можно использовать для нескольких кораблей?**
О: Да! Создайте несколько config файлов (как `carrack.yaml`, `idris.yaml`) с разными URL, затем запустите: `specter.exe --config carrack.yaml`

**В: Программа говорит "rate limited" - что это значит?**
О: Сервер занят и попросил нас замедлиться. Программа автоматически ждет 50-150мс (вместо 5-20мс) перед следующей попыткой. Это нормально во время загруженных продаж!

**В: Какой часовой пояс использовать для режима по расписанию?**
О: Всегда используйте **UTC время** и заканчивайте на `Z`. Сначала конвертируйте ваше местное время в UTC используя worldtimebuddy.com или Google.

### Решение проблем

**"No item URL specified"**
- Вы забыли вставить URL корабля в config.yaml ИЛИ забыли использовать флаг --url
- Файл config.yaml включен в загрузку - убедитесь что вы извлекли весь ZIP

**"Failed to launch browser"**
- Убедитесь что Chrome установлен (настоятельно рекомендуется)
- Попробуйте удалить папку `.specter` в вашей домашней директории и войдите снова

**"macOS Security Warning" или "iTerm запретил приложению изменять файлы" (Mac)**
Это функция безопасности macOS которая предотвращает приложения терминала от создания файлов в определенных местах:

**Лучшее решение: Предоставить Terminal полный доступ к диску**
1. Откройте **Системные настройки** (или System Preferences на старых macOS)
2. Перейдите в **Конфиденциальность и безопасность** → **Полный доступ к диску**
3. Нажмите на значок замка и введите пароль
4. Нажмите кнопку **+** и добавьте приложение терминала:
   - Если используете iTerm: Выберите **iTerm.app** из Программы
   - Если используете Terminal: Выберите **Terminal.app** из Программы/Утилиты
5. Включите чекбокс рядом с приложением терминала
6. **Перезапустите приложение терминала** (полностью закройте и откройте снова)
7. Попробуйте запустить Specter снова

**Альтернатива: Используйте Terminal.app вместо iTerm**
- Встроенный Terminal.app часто имеет необходимые разрешения по умолчанию
- Откройте Terminal.app из Программы/Утилиты
- Перейдите в папку Specter и запустите `./specter`

**"Chrome is already running" или ошибка "ProcessSingleton" / "SingletonLock"**
Это происходит когда Chrome уже открыт и использует тот же профиль:

1. **Полностью закройте ВСЕ окна Chrome**
   - Windows: Проверьте Диспетчер задач → Завершите все процессы Chrome.exe
   - Mac: Проверьте Мониторинг системы → Завершите все процессы Chrome
   - Или в Mac Terminal: `killall 'Google Chrome'`

2. **Попробуйте запустить Specter снова**
   - Specter запустит Chrome со своим изолированным профилем
   - Ваши существующие данные Chrome не будут затронуты

**"Session expired"**
- Ваш вход истек
- Запустите программу снова и она откроет Chrome для входа

**Ошибка "Insufficient store credit available"**
- У вас недостаточно store credit в аккаунте RSI чтобы покрыть стоимость корабля
- Проверьте баланс store credit на robertsspaceindustries.com
- Добавьте больше store credit на ваш аккаунт или выберите менее дорогой корабль
- Приложение работает ТОЛЬКО со store credit - вы не можете использовать кредитные карты или PayPal

**"Invalid sale start time format"**
- Убедитесь что используете правильный формат: `YYYY-MM-DDTHH:MM:SSZ`
- Всегда заканчивайте на `Z` для UTC времени
- Пример: `2025-01-15T23:00:00Z`

**Программа сразу закрывается на Windows**
- Возможно нужно разрешить ее в Windows Defender
- Правый клик на specter.exe → Свойства → Разблокировать → Применить

**"Sale window has already passed"**
- Указанное вами время уже прошло
- Проверьте конвертацию времени (убедитесь что использовали UTC, а не местное время)
- Убедитесь что дата правильная

**Антивирус блокирует/помечает программу (Kaspersky, Windows Defender, и др.)**

Это **ложное срабатывание** - крайне распространенная проблема для инструментов автоматизации браузера. Вот почему:

**Почему это происходит?**
- Программа запускает браузеры, управляет ими удаленно и делает сетевые запросы
- Антивирусное ПО не может отличить легитимную автоматизацию от вредоносной
- Программа НЕ имеет цифровой подписи (требует дорогой сертификат ~$300/год)
- Частые детекции: "PDM:Trojan.Bazon.a", "Trojan:Win32/Wacatac", или похожие

**Программа безопасна:**
- ✅ 100% открытый исходный код - вы можете проверить весь код на GitHub
- ✅ Не собирает данные - все работает локально
- ✅ Нет сетевого доступа кроме официального сайта RSI
- ✅ Сборки автоматизированы через GitHub Actions (видимы в репозитории)

**Как исправить:**
1. **Kaspersky:** Настройки → Угрозы и исключения → Управление исключениями → Добавить → Выберите `specter.exe`
2. **Windows Defender:** Безопасность Windows → Защита от вирусов и угроз → Управление настройками → Исключения → Добавить исключение → Файл → Выберите `specter.exe`
3. **Другие антивирусы:** Ищите "Исключения", "Белый список", или "Доверенные приложения" в настройках

**Все еще беспокоитесь?**
- Проверьте исходный код самостоятельно на GitHub
- Соберите из исходников вместо использования готовых бинарников
- Проверьте хеш файла с опубликованным на странице релизов
- Запустите в виртуальной машине если хотите дополнительную изоляцию

### Поддержка

Нужна помощь? Проверьте страницу issues на GitHub или спросите в сообществе Star Citizen.

### Правовая оговорка

⚠️ **Используйте на свой риск.** Этот инструмент может нарушать Условия использования RSI. Авторы не несут ответственности за любые последствия включая блокировку аккаунта. Это программное обеспечение предоставляется "как есть" без гарантий.

Этот инструмент не собирает никакие данные. Все работает локально на вашем компьютере. Ваш пароль RSI обрабатывается только Chrome, никогда этой программой.

---

**Good luck with your ship hunt! / Удачи в охоте за кораблем!** 🚀
