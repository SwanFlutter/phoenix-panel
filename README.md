# 🔥 PHOENIX PANEL

پنل مدیریت سابسکریپشن پروکسی چند-پروتکلی. مدیریت کاربران، نود‌ها و inbound‌ها روی
**Xray-core** و **sing-box** از یک API، همراه با تولید لینک سابسکریپشن شخصی برای هر کاربر
(VLESS, VMess, Trojan, Shadowsocks, Hysteria2, TUIC).

**مخزن:** [github.com/SwanFlutter/phoenix-panel](https://github.com/SwanFlutter/phoenix-panel)

> **وضعیت: فاز پایه.** این مخزن شامل backend تولیدی است — config، database، auth، security middleware،
> abstraction پروکسی کور، تولید لینک/سابسکریپشن، و REST API. آداپتورهای پروکسی کور
> (`internal/core/xray.go`, `internal/core/singbox.go`) به‌صورت stub در مرز gRPC/HTTP
> متصل شده‌اند. فرانت‌اند React و OpenAPI spec مراحل بعدی هستند. مراجعه کنید به [Roadmap](#roadmap).

---

## فهرست مطالب

- [معماری](#معماری)
- [نصب](#نصب)
  - [پیش‌نیازها](#پیشنیازها)
  - [روش ۱: Docker — یک دستور، نصب کامل](#روش-۱-docker--یک-دستور-نصب-کامل-)
  - [روش ۲: نصب Native بدون Docker](#روش-۲-نصب-native-بدون-docker)
- [حذف پنل](#حذف-پنل)
- [تنظیمات](#تنظیمات)
- [API](#api)
- [امنیت](#امنیت)
- [عیب‌یابی](#عیبیابی)
- [Roadmap](#roadmap)

---

## معماری

```
cmd/phoenix            — نقطه ورود سرور (config → db → router → http)
internal/
  config              — تنظیمات از env + اعتبارسنجی
  models              — مدل‌های GORM + انواع دامنه (User, Node, Inbound, …)
  database            — اتصال (SQLite/Postgres)، migrate، seed
  security            — هش argon2id، JWT، تولید token/uuid
  middleware          — auth، rate limiting، CORS، security headers، logging
  audit               — لاگ audit امنیتی
  core                — رابط ProxyCore + آداپتورهای Xray/sing-box
  links               — تولید share-URL و سند سابسکریپشن
  service             — منطق تجاری (users، auth، nodes، subscriptions)
  api                 — هندلرهای Gin، DTO‌ها، router
migrations            — schema SQL کانونیکال (PostgreSQL dialect)
```

لایه‌بندی سختگیرانه است: `api` → `service` → `models`/`database`.
هندلرها مستقیماً با DB کار نمی‌کنند؛ سرویس‌ها تمام invariant‌ها و تراکنش‌ها را مدیریت می‌کنند.

---

## نصب

### پیش‌نیازها

| روش | نیازمندی‌ها | سختی | سرعت |
|-----|------------|------|------|
| **Docker** ✅ پیشنهادی | فقط اینترنت + دسترسی root | آسان | سریع |
| **Native** | Linux/Ubuntu، systemd | متوسط | معمولی |

#### نیازمندی‌های سیستم

| مورد | حداقل | پیشنهادی |
|------|-------|----------|
| سیستم‌عامل | Ubuntu 20.04 / Debian 11 | Ubuntu 22.04 LTS |
| CPU | 1 هسته | 2 هسته |
| RAM | 512 MB | 1 GB |
| دیسک | 2 GB | 10 GB |
| پورت | 8080 (قابل تغییر) | 80/443 با reverse proxy |

> 💡 اسکریپت نصب فقط روی **Linux x86_64** و **ARM64** تست شده. Windows و macOS پشتیبانی نمی‌شوند.

---

### روش ۱: Docker — نصب کامل با یک دستور ✅

تمام مراحل (Git، Docker، کلون مخزن، تنظیمات، build، راه‌اندازی) به صورت **خودکار** انجام می‌شود.

#### مرحله ۱ — وارد شدن به سرور

```bash
ssh root@YOUR_SERVER_IP
# یا با sudo:
sudo -i
```

#### مرحله ۲ — اجرای اسکریپت نصب

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) docker
```

> ⚠️ **نیاز به دسترسی root دارد.** اگر با کاربر معمولی هستید، ابتدا `sudo -i` بزنید.

#### مرحله ۳ — وارد کردن پسورد ادمین

اسکریپت پس از دانلود کد، این صفحه را نشان می‌دهد:

```
  ╔══════════════════════════════════════════════════════╗
  ║          تنظیم اولیه پنل فونیکس                     ║
  ╚══════════════════════════════════════════════════════╝

  🔑 پسورد ادمین (حداقل ۸ کاراکتر):
```

- پسورد را تایپ کنید (چیزی نمایش داده نمی‌شود — این طبیعی است)
- Enter بزنید
- اگر پسورد کمتر از ۸ کاراکتر باشد دوباره می‌پرسد

#### مرحله ۴ — انتخاب آدرس پنل

```
  🌐  آدرس پایه پنل (برای لینک‌های سابسکریپشن استفاده می‌شود)

    [1]  دامنه دارم و SSL دارد    (مثال: https://vpn.example.com)
    [2]  ندارم — از IP و پورت پیش‌فرض استفاده کن  (http://1.2.3.4:8080)

  انتخاب خود را وارد کنید [1/2]:
```

**گزینه ۱ — دامنه با SSL:**
```
  آدرس دامنه خود را وارد کنید (مثال: https://vpn.example.com): https://vpn.mydomain.com
```
- آدرس باید با `https://` شروع شود
- قبل از نصب مطمئن شوید DNS دامنه به IP سرور اشاره می‌کند

**گزینه ۲ — بدون دامنه:**
- آدرس به صورت `http://IP_SERVER:8080` به طور خودکار تنظیم می‌شود
- نیازی به وارد کردن چیزی نیست

#### مرحله ۵ — build و راه‌اندازی خودکار

اسکریپت بقیه کار را انجام می‌دهد:

```
[phoenix] ساخت image و راه‌اندازی با docker compose...
 ✔ Container phoenix-panel-postgres-1  Started
 ✔ Container phoenix-panel-panel-1     Started
[phoenix] منتظر راه‌اندازی سرویس...
```

این مرحله بسته به سرعت اینترنت **۱ تا ۵ دقیقه** طول می‌کشد.

#### مرحله ۶ — دریافت اطلاعات دسترسی

پس از راه‌اندازی موفق، این خلاصه نمایش داده می‌شود:

```
╔══════════════════════════════════════════════════════════╗
║        🎉  پنل فونیکس با موفقیت نصب شد                  ║
╚══════════════════════════════════════════════════════════╝

  ┌─ اطلاعات دسترسی ──────────────────────────────────────┐
  │  آدرس پنل:            http://1.2.3.4:8080
  │  لاگین ادمین:          http://1.2.3.4:8080/api/admin/login
  │  بررسی سلامت:          http://1.2.3.4:8080/healthz
  ├─ اطلاعات ورود ───────────────────────────────────────┤
  │  نام کاربری:           admin
  │  پسورد:                MySecurePass123
  ├─ دستورات مفید ───────────────────────────────────────┤
  │  مشاهده لاگ:           docker compose ... logs -f panel
  │  ری‌استارت:             docker compose ... restart panel
  │  توقف:                 docker compose ... down
  │  آپدیت:                bash <(curl -fsSL ...) docker
  └───────────────────────────────────────────────────────┘

  ⚠  این اطلاعات را ذخیره کنید — پسورد دیگر نمایش داده نخواهد شد.
```

> 🔴 **این اطلاعات را همین جا کپی کنید.** پسورد بعداً قابل مشاهده نیست.

#### مرحله ۷ — تأیید نصب

```bash
# بررسی سلامت سرویس
curl http://localhost:8080/healthz
# خروجی: {"status":"ok"}

# بررسی وضعیت کانتینرها
docker ps | grep phoenix
# باید دو کانتینر panel و postgres در حال اجرا باشند
```

#### دستورات مدیریت Docker

```bash
# مشاهده لاگ‌های زنده
docker compose -f /opt/phoenix-panel-src/docker-compose.yml logs -f panel

# ری‌استارت سرویس
docker compose -f /opt/phoenix-panel-src/docker-compose.yml restart panel

# توقف کامل
docker compose -f /opt/phoenix-panel-src/docker-compose.yml down

# توقف و حذف داده‌ها (⚠️ برگشت‌ناپذیر)
docker compose -f /opt/phoenix-panel-src/docker-compose.yml down --volumes

# آپدیت به آخرین نسخه
bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) docker
```

---

### روش ۲: نصب Native (بدون Docker)

باینری Go مستقیماً روی سرور build شده و به‌عنوان **سرویس systemd** نصب می‌شود.
مناسب برای سرورهایی که Docker ندارند یا می‌خواهید overhead کمتری داشته باشید.

#### مرحله ۱ — وارد شدن به سرور

```bash
ssh root@YOUR_SERVER_IP
sudo -i
```

#### مرحله ۲ — اجرای اسکریپت نصب

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) native
```

#### مرحله ۳ — وارد کردن تنظیمات

دقیقاً مانند روش Docker — پسورد ادمین و آدرس پنل پرسیده می‌شود (مراحل ۳ و ۴ بالا).

#### مرحله ۴ — build خودکار باینری

اسکریپت Go 1.22 را نصب و باینری را build می‌کند:

```
[phoenix] نصب Go 1.22.5...
[phoenix] ساخت باینری phoenix...
[phoenix] ساخت کامل شد: /opt/phoenix-panel-src/bin/phoenix
```

این مرحله **۳ تا ۱۰ دقیقه** طول می‌کشد (بسته به سرعت اینترنت و CPU).

#### مرحله ۵ — نصب و فعال‌سازی سرویس

```
[phoenix] نصب systemd unit...
● phoenix-panel.service - PHOENIX PANEL
     Loaded: loaded (/etc/systemd/system/phoenix-panel.service)
     Active: active (running)
```

#### دستورات مدیریت Native

```bash
# وضعیت سرویس
systemctl status phoenix-panel

# مشاهده لاگ‌های زنده
journalctl -u phoenix-panel -f

# ری‌استارت
systemctl restart phoenix-panel

# توقف
systemctl stop phoenix-panel

# شروع دوباره
systemctl start phoenix-panel

# غیرفعال کردن autostart
systemctl disable phoenix-panel

# ویرایش تنظیمات
nano /opt/phoenix/.env
# بعد از ویرایش حتماً restart کنید:
systemctl restart phoenix-panel
```

#### مسیرهای فایل‌ها (Native)

| فایل | مسیر |
|------|------|
| باینری | `/opt/phoenix/phoenix` |
| تنظیمات | `/opt/phoenix/.env` |
| دیتابیس SQLite | `/opt/phoenix/data/phoenix.db` |
| لاگ سرویس | `journalctl -u phoenix-panel` |
| سرویس systemd | `/etc/systemd/system/phoenix-panel.service` |

---

## حذف پنل

برای حذف **کامل** پنل، تمام داده‌ها، کانتینرها و فایل‌های نصب:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) uninstall
```

#### چه اتفاقی می‌افتد؟

ابتدا لیست کامل موارد حذف‌شدنی نمایش داده می‌شود:

```
⚠️   حذف پنل فونیکس
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  این عملیات موارد زیر را حذف می‌کند:
    • سرویس systemd  (phoenix-panel)
    • کانتینرهای Docker و image پنل
    • فایل‌های نصب   (/opt/phoenix)
    • سورس کد        (/opt/phoenix-panel-src)
    • داده‌ها و دیتابیس  ⚠️  غیرقابل بازگشت

  برای ادامه 'YES' تایپ کنید (برای انصراف هر چیز دیگری):
```

- برای **تأیید** دقیقاً `YES` (با حروف بزرگ) تایپ کنید
- برای **انصراف** هر کلید دیگری (مثلاً Enter یا `no`) کافی است

پس از تأیید، به ترتیب انجام می‌شود:

1. سرویس systemd متوقف و حذف می‌شود
2. کانتینرهای Docker و volume‌ها پاک می‌شوند
3. Docker image پنل حذف می‌شود
4. پوشه `/opt/phoenix` حذف می‌شود
5. سورس کد `/opt/phoenix-panel-src` حذف می‌شود
6. کاربر سیستمی `phoenix` حذف می‌شود

```
╔══════════════════════════════════════════════════════════╗
║        🗑️   پنل فونیکس کاملاً حذف شد                    ║
╚══════════════════════════════════════════════════════════╝
```

> ⚠️ **هشدار:** تمام داده‌های کاربران، تنظیمات و دیتابیس پس از حذف قابل بازیابی **نیستند**. قبل از حذف از داده‌های مهم backup بگیرید.

#### نصب مجدد پس از حذف

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) docker
```

---

## تنظیمات

تمام تنظیمات از طریق متغیرهای محیطی (`/opt/phoenix-panel-src/.env`) انجام می‌شود.
فایل `.env.example` لیست کامل را دارد.

### تنظیمات اصلی

| متغیر | پیش‌فرض | اجباری | توضیح |
|-------|---------|--------|-------|
| `PHOENIX_JWT_SECRET` | — | ✅ | کلید JWT (حداقل ۳۲ کاراکتر). تولید: `openssl rand -hex 32` |
| `PHOENIX_ADMIN_PASSWORD` | — | ✅ | پسورد ادمین (در اولین اجرا ساخته می‌شود) |
| `PHOENIX_BASE_URL` | `http://localhost:8080` | ⚠️ | آدرس عمومی برای لینک‌های سابسکریپشن |
| `PHOENIX_DB_DRIVER` | `sqlite` | — | `sqlite` یا `postgres` |
| `PHOENIX_PORT` | `8080` | — | پورت listen |
| `PHOENIX_MODE` | `release` | — | `debug` یا `release` |

### تنظیمات دیتابیس

#### SQLite (پیش‌فرض)

```bash
PHOENIX_DB_DRIVER=sqlite
PHOENIX_DB_SQLITE_PATH=./data/phoenix.db
```

#### PostgreSQL

```bash
PHOENIX_DB_DRIVER=postgres
PHOENIX_DB_HOST=localhost
PHOENIX_DB_PORT=5432
PHOENIX_DB_USER=phoenix
PHOENIX_DB_PASSWORD=your-secure-password
PHOENIX_DB_NAME=phoenix
```

### تنظیمات امنیتی

```bash
# کلید JWT
PHOENIX_JWT_SECRET=your-32-char-secret-here

# مدت اعتبار توکن
PHOENIX_JWT_TTL=24h

# Rate limiting
PHOENIX_RATE_RPS=20          # درخواست در ثانیه به ازای هر IP
PHOENIX_LOGIN_RATE_RPS=1     # تلاش login در ثانیه به ازای هر IP

# CORS
PHOENIX_CORS_ORIGINS=*       # یا دامنه‌های خاص با کاما جدا شده
```

---

## API

برای endpoint‌های ادمین احراز هویت JWT لازم است. endpoint‌های عمومی (health، سابسکریپشن) بدون احراز هویت در دسترس هستند.

### احراز هویت

| متد | مسیر | Auth | توضیح |
|-----|------|------|-------|
| `POST` | `/api/admin/login` | — | دریافت توکن JWT |
| `GET` | `/api/admin/me` | Bearer | اطلاعات ادمین جاری |
| `POST` | `/api/admin/change-password` | Bearer | تغییر پسورد ادمین |

### مدیریت کاربران

| متد | مسیر | Auth | توضیح |
|-----|------|------|-------|
| `GET` | `/api/admin/users` | Bearer | لیست تمام کاربران |
| `POST` | `/api/admin/users` | Bearer | ساخت کاربر جدید |
| `GET` | `/api/admin/users/:id` | Bearer | جزئیات کاربر |
| `PATCH` | `/api/admin/users/:id` | Bearer | ویرایش کاربر |
| `DELETE` | `/api/admin/users/:id` | Bearer | حذف کاربر |
| `POST` | `/api/admin/users/:id/reset` | Bearer | ریست ترافیک کاربر |
| `POST` | `/api/admin/users/:id/regenerate-sub` | Bearer | تجدید توکن سابسکریپشن |

### مدیریت نود و Inbound

| متد | مسیر | Auth | توضیح |
|-----|------|------|-------|
| `GET` | `/api/admin/nodes` | Bearer | لیست نودها |
| `POST` | `/api/admin/nodes` | Bearer (sudo) | ساخت نود |
| `DELETE` | `/api/admin/nodes/:id` | Bearer (sudo) | حذف نود |
| `POST` | `/api/admin/inbounds` | Bearer (sudo) | ساخت inbound |
| `DELETE` | `/api/admin/inbounds/:id` | Bearer (sudo) | حذف inbound |

### endpoint‌های عمومی

| متد | مسیر | Auth | توضیح |
|-----|------|------|-------|
| `GET` | `/sub/:token` | token | دریافت سند سابسکریپشن |
| `GET` | `/healthz` | — | بررسی سلامت |
| `GET` | `/readyz` | — | بررسی آمادگی |

### مثال‌های استفاده

#### ورود و دریافت توکن

```bash
TOKEN=$(curl -s http://localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "admin",
    "password": "your-admin-password"
  }' | jq -r .token)
```

#### ساخت کاربر

```bash
curl -s http://localhost:8080/api/admin/users \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "alice",
    "data_limit": 53687091200
  }' | jq
```

#### دریافت سابسکریپشن

```bash
curl -s http://localhost:8080/sub/YOUR_SUB_TOKEN
```

---

## امنیت

- **پسوردها:** هش Argon2id (64 MiB، t=3، p=2)، بررسی constant-time
- **توکن‌ها:** JWT با HS256 و algorithm pinning؛ توکن‌های سابسکریپشن ۲۴ بایتی غیرقابل حدس
- **محافظت Brute Force:** rate limit سختگیرانه روی `/api/admin/login`
- **Security Headers:** CSP، HSTS، X-Frame-Options، nosniff
- **Audit Trail:** تمام عملیات privileged در `audit_logs` ثبت می‌شود
- **Least Privilege:** نقش‌های sudo vs admin؛ تغییر نود/inbound نیاز به sudo دارد

---

## عیب‌یابی

### کانتینر راه‌اندازی نمی‌شود

```bash
# مشاهده لاگ‌ها
docker compose logs panel

# مشکلات رایج:
# ۱. پورت در حال استفاده است
#    راه‌حل: PHOENIX_PORT را در .env تغییر دهید

# ۲. فایل .env وجود ندارد
#    راه‌حل: cp .env.example .env && nano .env

# ۳. خطای ساخت cmd/phoenix
#    راه‌حل: مطمئن شوید آخرین کد را دارید (git pull)
```

### پنل قابل دسترس نیست

```bash
# بررسی وضعیت سرویس
docker ps | grep phoenix

# بررسی پورت
netstat -tlnp | grep 8080

# تست health endpoint
curl http://localhost:8080/healthz

# اگر از خارج قابل دسترس نیست:
# PHOENIX_BASE_URL=http://your-server-ip:8080 را در .env تنظیم کنید
```

### خطای دیتابیس

```bash
# SQLite: بررسی دسترسی فایل
ls -la ./data/phoenix.db

# PostgreSQL: بررسی اتصال
psql -h localhost -U phoenix -d phoenix

# ریست دیتابیس (⚠️ تمام داده‌ها حذف می‌شوند)
rm ./data/phoenix.db   # فقط SQLite
```

### ورود به پنل ممکن نیست

```bash
# بررسی پسورد ادمین در .env
grep PHOENIX_ADMIN_PASSWORD /opt/phoenix-panel-src/.env

# مشاهده لاگ‌های login
docker compose logs panel | grep -i login
```

### حذف و نصب مجدد

```bash
# حذف کامل
sudo bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) uninstall

# نصب مجدد
sudo bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) docker
```

---

## توسعه

```bash
# نصب وابستگی‌ها
make tidy

# اجرای تست‌ها
make test

# اجرای linter
make lint

# اجرای محلی
make run
```

### ساختار پروژه

```
cmd/phoenix/        — نقطه ورود اصلی
internal/           — منطق تجاری و سرویس‌ها
migrations/         — فایل‌های migration دیتابیس
docker-compose.yml  — تنظیمات Docker Compose
Dockerfile          — تعریف Docker image
install.sh          — اسکریپت نصب خودکار
```

---

## Roadmap

- [ ] اتصال واقعی Xray gRPC + sing-box در `internal/core`
- [ ] scheduler جمع‌آوری ترافیک + تطبیق مصرف به ازای هر کاربر
- [ ] داشبورد ادمین React + TypeScript و پنل کاربری (`/web`)
- [ ] OpenAPI 3.1 spec + مستندات تولیدشده
- [ ] راهنمای ادمین و کاربر
- [ ] Kubernetes manifests و Helm charts

---

## لینک‌های مفید

- **مخزن اصلی:** [github.com/SwanFlutter/phoenix-panel](https://github.com/SwanFlutter/phoenix-panel)
- **اسکریپت نصب:** [install.sh](https://github.com/SwanFlutter/phoenix-panel/blob/main/install.sh)
- **گزارش مشکل:** [GitHub Issues](https://github.com/SwanFlutter/phoenix-panel/issues)

---

## License

در انتظار تصمیم صاحب پروژه.

---

**آخرین به‌روزرسانی:** ۱۴۰۵/۰۳/۳۰  
**مخزن:** [SwanFlutter/phoenix-panel](https://github.com/SwanFlutter/phoenix-panel)
