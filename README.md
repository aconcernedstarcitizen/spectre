# Specter - RSI Store Automated Checkout

**Lightning-fast automated checkout for limited Star Citizen ship sales**

[English](#english) | [Русский](#русский)

---

## English

### What is This?

Specter is a tool that automatically buys limited-edition ships from the Star Citizen store (robertsspaceindustries.com) at lightning speed. When ships sell out in seconds, this gives you the best chance to complete your purchase.

**Key Features:**
- ⚡ Ultra-fast API-based checkout (under 1 second!)
- 🔄 Smart retry system - keeps trying for 5 minutes if the ship isn't available yet
- 💳 Automatic store credit application
- 🤖 Rate limit detection - automatically backs off if the server is busy
- 🎯 Launch 5 minutes early - it will wait and keep trying once the sale starts

### Requirements

**What You Need:**
- A computer (Windows or Mac)
- Google Chrome browser installed
- A Star Citizen account with store credit
- Basic computer skills (opening files, running programs)

### Installation

#### For Windows:

1. **Download Specter:**
   - Download `specter.exe` from the releases page
   - Save it to a folder (like `C:\Specter`)

2. **Make sure Chrome is installed:**
   - If you don't have Chrome, download it from google.com/chrome

#### For Mac:

1. **Download Specter:**
   - Download `specter` (Mac version) from the releases page
   - Save it to a folder (like `/Users/YourName/Specter`)

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

5. **Keep the Chrome window open** - the program will show an error and exit. This is expected!

6. Your login is now saved! Close everything.

#### Step 2: Configure the Ship URL

1. **Find config.yaml** in the Specter folder
   - **Important:** The file must be named exactly `config.yaml` (not `config.yaml.example` or anything else)
   - If you see `config.yaml.example`, rename it to `config.yaml` before proceeding

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

### How to Use During a Sale

**Launch the program 5 minutes BEFORE the sale starts!**

#### On Windows:

1. Open Command Prompt
2. Go to the Specter folder: `cd C:\Specter`
3. Run: `specter.exe`
4. Wait! The program will keep trying until the ship becomes available

#### On Mac:

1. Open Terminal
2. Go to the Specter folder: `cd /Users/YourName/Specter`
3. Run: `./specter`
4. Wait! The program will keep trying until the ship becomes available

**What You'll See:**

```
⏱️  Will retry for up to 300 seconds if item not available
⚠️  Attempt 1 failed - retrying in 73ms (remaining: 4m59s)...
⚠️  Attempt 2 failed - retrying in 91ms (remaining: 4m58s)...
✓ Added to cart successfully!
🎉 Success after 15 attempt(s) in 5.2s
✓ ORDER COMPLETED!
```

The program will automatically:
1. Keep trying to add the ship to your cart (until the sale starts)
2. Apply your store credit
3. Complete the checkout
4. Tell you when it's done!

### Settings You Can Change

Open `config.yaml` to customize:

#### How Long to Keep Trying:
```yaml
retry_duration_seconds: 300  # Default is 5 minutes (300 seconds)
```
Want to try for longer? Change this to 600 (10 minutes) or more.

#### Use Store Credit:
```yaml
auto_apply_credit: true  # Set to false if you want to pay with credit card
```

#### Test Mode (Practice Without Buying):
```yaml
dry_run: true  # Set to true to practice, false for real purchases
```

### Common Questions

**Q: Will this get me banned?**
A: Using automation tools may violate RSI's Terms of Service. Use at your own risk. This tool is designed to be respectful (human-like delays, rate limit detection), but there's always a risk.

**Q: How fast is it?**
A: The checkout process takes less than 1 second once the ship is in your cart. The retry system will keep trying for 5 minutes (or however long you configure) before that.

**Q: Do I need programming experience?**
A: No! Just follow the instructions above. If you can open files and type commands, you can use this.

**Q: What if it doesn't work?**
A: Make sure you:
- Logged in successfully (Step 1)
- Put the correct ship URL in config.yaml
- Have enough store credit in your account
- Have a fast internet connection

**Q: Can I use this for multiple ships?**
A: Yes! Create multiple config files (like `carrack.yaml`, `idris.yaml`) with different URLs, then run: `specter.exe --config carrack.yaml`

**Q: The program says "rate limited" - what does that mean?**
A: The server is busy and asked us to slow down. The program automatically waits longer (2-5 seconds) before trying again. This is normal during busy sales!

### Troubleshooting

**"No item URL specified"**
- You forgot to put the ship URL in config.yaml
- Or the config.yaml file has an error
- Make sure the config file is named exactly `config.yaml` (not `config.yaml.example`)

**"Failed to launch browser"**
- Make sure Chrome is installed
- Try deleting the `.specter` folder in your home directory and login again

**"Session expired"**
- Your login expired
- Run the program again and it will open Chrome for you to login

**Program exits immediately on Windows**
- You might need to allow it through Windows Defender
- Right-click specter.exe → Properties → Unblock → Apply

### Support

Need help? Check the issues page on GitHub or ask in the Star Citizen community.

### Legal Disclaimer

⚠️ **Use at your own risk.** This tool may violate RSI's Terms of Service. The authors are not responsible for any consequences including account suspension. This software is provided "as is" without warranty.

This tool does not collect any data. Everything runs locally on your computer. Your RSI password is handled only by Chrome, never by this program.

---

## Русский

### Что это?

Specter - это инструмент, который автоматически покупает лимитированные корабли из магазина Star Citizen (robertsspaceindustries.com) с молниеносной скоростью. Когда корабли распродаются за секунды, это дает вам лучший шанс завершить покупку.

**Основные возможности:**
- ⚡ Сверхбыстрое оформление через API (меньше 1 секунды!)
- 🔄 Умная система повторов - продолжает попытки 5 минут, если корабль еще недоступен
- 💳 Автоматическое применение store credit
- 🤖 Определение ограничения запросов - автоматически замедляется, если сервер занят
- 🎯 Запуск за 5 минут до начала - будет ждать и пытаться, когда начнется продажа

### Требования

**Что вам нужно:**
- Компьютер (Windows или Mac)
- Установленный браузер Google Chrome
- Аккаунт Star Citizen со store credit
- Базовые навыки работы с компьютером (открытие файлов, запуск программ)

### Установка

#### Для Windows:

1. **Скачайте Specter:**
   - Скачайте `specter.exe` со страницы релизов
   - Сохраните в папку (например `C:\Specter`)

2. **Убедитесь что Chrome установлен:**
   - Если у вас нет Chrome, скачайте его с google.com/chrome

#### Для Mac:

1. **Скачайте Specter:**
   - Скачайте `specter` (версия для Mac) со страницы релизов
   - Сохраните в папку (например `/Users/ВашеИмя/Specter`)

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

5. **Оставьте окно Chrome открытым** - программа покажет ошибку и закроется. Это ожидаемо!

6. Ваш вход теперь сохранен! Закройте все.

#### Шаг 2: Настройте URL корабля

1. **Найдите config.yaml** в папке Specter
   - **Важно:** Файл должен называться точно `config.yaml` (не `config.yaml.example` или что-то другое)
   - Если вы видите `config.yaml.example`, переименуйте его в `config.yaml` перед продолжением

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

### Как использовать во время продажи

**Запустите программу за 5 минут ДО начала продажи!**

#### На Windows:

1. Откройте Командную строку
2. Перейдите в папку Specter: `cd C:\Specter`
3. Запустите: `specter.exe`
4. Ждите! Программа будет продолжать попытки пока корабль не станет доступен

#### На Mac:

1. Откройте Terminal
2. Перейдите в папку Specter: `cd /Users/ВашеИмя/Specter`
3. Запустите: `./specter`
4. Ждите! Программа будет продолжать попытки пока корабль не станет доступен

**Что вы увидите:**

```
⏱️  Will retry for up to 300 seconds if item not available
⚠️  Attempt 1 failed - retrying in 73ms (remaining: 4m59s)...
⚠️  Attempt 2 failed - retrying in 91ms (remaining: 4m58s)...
✓ Added to cart successfully!
🎉 Success after 15 attempt(s) in 5.2s
✓ ORDER COMPLETED!
```

Программа автоматически:
1. Будет пытаться добавить корабль в корзину (пока не начнется продажа)
2. Применит ваш store credit
3. Завершит оформление
4. Сообщит вам когда все готово!

### Настройки которые можно изменить

Откройте `config.yaml` для настройки:

#### Как долго продолжать попытки:
```yaml
retry_duration_seconds: 300  # По умолчанию 5 минут (300 секунд)
```
Хотите пытаться дольше? Измените на 600 (10 минут) или больше.

#### Использовать Store Credit:
```yaml
auto_apply_credit: true  # Установите false если хотите платить кредитной картой
```

#### Тестовый режим (тренировка без покупки):
```yaml
dry_run: true  # Установите true для тренировки, false для реальных покупок
```

### Частые вопросы

**В: Меня забанят за это?**
О: Использование инструментов автоматизации может нарушать Условия использования RSI. Используйте на свой риск. Этот инструмент разработан быть уважительным (человекоподобные задержки, определение ограничений), но риск всегда есть.

**В: Насколько это быстро?**
О: Процесс оформления занимает меньше 1 секунды после того как корабль в корзине. Система повторов будет пытаться 5 минут (или сколько настроите) до этого.

**В: Нужен ли опыт программирования?**
О: Нет! Просто следуйте инструкциям выше. Если вы можете открывать файлы и вводить команды, вы можете это использовать.

**В: Что если не работает?**
О: Убедитесь что вы:
- Успешно вошли (Шаг 1)
- Вставили правильный URL корабля в config.yaml
- Имеете достаточно store credit в аккаунте
- Имеете быстрое интернет-соединение

**В: Можно использовать для нескольких кораблей?**
О: Да! Создайте несколько config файлов (как `carrack.yaml`, `idris.yaml`) с разными URL, затем запустите: `specter.exe --config carrack.yaml`

**В: Программа говорит "rate limited" - что это значит?**
О: Сервер занят и попросил нас замедлиться. Программа автоматически ждет дольше (2-5 секунд) перед следующей попыткой. Это нормально во время загруженных продаж!

### Решение проблем

**"No item URL specified"**
- Вы забыли вставить URL корабля в config.yaml
- Или в файле config.yaml есть ошибка
- Убедитесь что файл называется точно `config.yaml` (не `config.yaml.example`)

**"Failed to launch browser"**
- Убедитесь что Chrome установлен
- Попробуйте удалить папку `.specter` в вашей домашней директории и войдите снова

**"Session expired"**
- Ваш вход истек
- Запустите программу снова и она откроет Chrome для входа

**Программа сразу закрывается на Windows**
- Возможно нужно разрешить ее в Windows Defender
- Правый клик на specter.exe → Свойства → Разблокировать → Применить

### Поддержка

Нужна помощь? Проверьте страницу issues на GitHub или спросите в сообществе Star Citizen.

### Правовая оговорка

⚠️ **Используйте на свой риск.** Этот инструмент может нарушать Условия использования RSI. Авторы не несут ответственности за любые последствия включая блокировку аккаунта. Это программное обеспечение предоставляется "как есть" без гарантий.

Этот инструмент не собирает никакие данные. Все работает локально на вашем компьютере. Ваш пароль RSI обрабатывается только Chrome, никогда этой программой.

---

**Good luck with your ship hunt! / Удачи в охоте за кораблем!** 🚀
