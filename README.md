# LayerInvoice 🚀

[![Build and Publish Docker Image](https://github.com/BasilSaeed/LayerInvoice/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/BasilSaeed/LayerInvoice/actions/workflows/docker-publish.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8.svg?style=flat&logo=go)](https://go.dev)
[![Database](https://img.shields.io/badge/Database-SQLite-003B57.svg?style=flat&logo=sqlite)](https://sqlite.org)
[![HTMX](https://img.shields.io/badge/HTMX-%E2%9A%A1%20Reactive-blue)](https://htmx.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**LayerInvoice** is a premium, self-hosted, multi-tenant CRM, ERP, and professional Invoice/Estimate builder purpose-built for **3D Printing Service Shops, makerspaces, and engineering freelancers**. It merges sleek document-centric design with powerful automation, database persistence, and an advanced 3D printing cost estimator.

---

## 🎨 Inspiration & Appreciation

LayerInvoice is heavily inspired by the visual design and document-centric layout of **[SimpleInvoice](https://github.com/RihanArfan/SimpleInvoice)**. We express our sincere appreciation to the SimpleInvoice creators; their layout served as the benchmark for designing a clean, high-contrast, premium, paper-like document editor. LayerInvoice builds upon that gorgeous document-first philosophy and extends it into a complete, Go-native CRM/ERP geared toward 3D fabrication.

---

## ✨ Features

- **💼 CRM Client Management**: Full client profiles with integrated billing/shipping addresses, contact emails, direct WhatsApp integration, and customizable client categorization.
- **📜 Estimates & Quotes CRM**: Draft fully items-costed quotes. Transition statuses, export professional PDFs, and convert quotes to full invoices with a single click.
- **🧾 Automated Invoice Builder**: Clean, inline, auto-saved invoice generation supporting custom line-items, discounts, and custom tax brackets.
- **⚡ Real-time 3D Printing Cost Calculator**: Computes exact production costs and margins live within the invoice and quote interfaces, removing manual math.
- **🧵 Filament Profiles Manager**: Register spools, custom material profiles (PLA, PETG, ABS, TPU, Carbon Fiber, Resins), and track real-time cost-per-gram properties.
- **📬 Debounced SMTP Email Dispatcher**: When an invoice is updated, a smart background worker waits for a 5-second silence period (debouncing keys) before compiling the updated PDF and emailing it to the client. Includes resilient SMTP error handling.
- **💸 Integrated Payment Register**: Log payments (Cash, Bank Transfer, PayPal, Stripe, etc.) against invoices. outstanding balances, earnings, and statistics update instantly.
- **📊 Interactive Financial Dashboard**: Track gross revenue, outstanding balances, payment distributions, and active invoice charts.
- **🔒 Multi-Tenant Engine**: Clean data separation between operating branches, organizations, or users.
- **💡 High-Contrast Modern Visual Theme**: Styled with a robust CSS layer atop PicoCSS, locked into high-contrast light mode to ensure optimal readability on both light and dark operating systems.

---

## 📐 3D Print Cost Calculator Math

LayerInvoice features an advanced cost breakdown engine to ensure 3D printing services maintain healthy profit margins. The calculator evaluates parameters in real-time using these formulas:

### 1. Material Costs
$$\text{Raw Filament Cost} = \text{Grams Weight} \times \text{Cost per Gram}$$
$$\text{Filament Profit Markup} = \text{Raw Filament Cost} \times \left( \frac{\text{Filament Profit \%}}{100} \right)$$
$$\text{Total Material Cost} = \text{Raw Filament Cost} + \text{Filament Profit Markup}$$

### 2. Operating & Labor Costs
$$\text{Machine Wear Cost} = \text{Print Hours} \times \text{Hourly Machine Rate}$$
$$\text{Labor Fee} = \text{Configured Flat Rate}$$

### 3. Base Production Costs
$$\text{Base Production Cost} = \text{Total Material Cost} + \text{Machine Wear Cost} + \text{Labor Fee} + \text{Electricity} + \text{Post-Processing} + \text{Packaging}$$

### 4. Risk Allowance (Failure Buffer)
3D prints can fail mid-run due to support issues, power outages, or warping. LayerInvoice handles this elegantly:
$$\text{Failure Risk Allowance} = \text{Base Production Cost} \times \left( \frac{\text{Failure Rate \%}}{100} \right)$$
$$\text{Total Production Cost} = \text{Base Production Cost} + \text{Failure Risk Allowance}$$

### 5. Final Selling Price
A final job profit multiplier is applied to the production total to cover business growth and overheads:
$$\text{Final Profit Markup} = \text{Total Production Cost} \times \left( \frac{\text{Job Profit Multiplier \%}}{100} \right)$$
$$\text{Estimated Customer Price} = \text{Total Production Cost} + \text{Final Profit Markup} + \text{Shipping Cost}$$

---

## 🛠️ Technology Stack

LayerInvoice is built on a modern, ultra-lightweight, and lightning-fast tech stack:

* **Backend Language**: **Go (Golang) 1.26+** for blazing-fast speed, static compilation, and zero-runtime dependency packaging.
* **HTTP Router**: **Go-Chi** — a lightweight, idiomatic router.
* **Database Engine**: **SQLite** operating in high-performance **WAL (Write-Ahead Logging)** mode. Provides local database speeds with ACID transactions.
* **Migrations Tool**: **Goose** — integrated directly into the binary to automatically run database migrations on startup.
* **Frontend Reactivity**: **HTMX** for high-performance, single-page-app reactivity (re-rendering DOM nodes on the server without heavy JavaScript frameworks).
* **UI Micro-interactions**: **Alpine.js** for handling client-side state, modal resets, and dropdown behaviors.
* **CSS Framework**: **PicoCSS** combined with custom **Vanilla CSS** tokens, ensuring high readability, responsive structures, and crisp contrast.

---

## 🐳 Docker Deployment (Portainer / Docker-Compose)

LayerInvoice is designed to be fully self-hosted. Using Docker ensures a uniform environment and easy updates.

### Persistent Local Storage & Zero Data Loss
LayerInvoice utilizes SQLite. To ensure your database, session records, and uploads are **never lost** when the container closes, updates, or reboots, you **must** mount the application's data directory to the host machine. 

SQLite creates side-car files (`layerinvoice.db-wal` and `layerinvoice.db-shm`) during active operations. Mounting the **entire parent folder** (e.g. `/app/data` inside the container) instead of mounting a single file preserves these transactional logs safely and prevents corruption.

### 📋 Standard `docker-compose.yml`

Create a folder for LayerInvoice (e.g., `layerinvoice`), place the following `docker-compose.yml` inside it, and customize the environment variables:

```yaml
version: "3.9"

services:
  app:
    image: ghcr.io/basilsaeed/layerinvoice:latest
    container_name: layerinvoice_app
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - DB_PATH=/app/data/layerinvoice.db
      - SESSION_SECRET=choose_a_long_secure_random_string_here_32_chars
      - APP_ENV=production
      - PORT=8080
      
      # Optional: SMTP Configuration for Auto-Email Invoices
      # - SMTP_HOST=smtp.gmail.com
      # - SMTP_PORT=587
      # - SMTP_USER=your-email@gmail.com
      # - SMTP_PASS=your-app-specific-password
      # - SMTP_FROM=billing@yourdomain.com
```

### 🚀 Running the Application

1. **Start the Stack**:
   Run the following command in the same directory:
   ```bash
   docker compose up -d
   ```
2. **Setup your Account**:
   Open `http://localhost:8080/setup` in your browser. Since the database starts empty, it will prompt you to create the initial Tenant organization and the Administrator user account.
3. **Login**:
   Access `http://localhost:8080/login` to begin managing your shop!

### ☸️ Portainer Deployment Instructions

Deploying via Portainer is extremely simple:
1. Log in to your **Portainer Dashboard**.
2. Navigate to **Stacks** ➡️ **Add Stack**.
3. Name your stack (e.g., `layerinvoice`).
4. Paste the `docker-compose.yml` content shown above into the Web Editor.
5. Under **Environment variables**, define your custom `SESSION_SECRET` and SMTP credentials.
6. Click **Deploy the stack**.
7. Ensure that the volume folder `./data` maps correctly on the host filesystem to secure persistent calculations.

---

## 💻 Local Development Setup

If you want to run the project from source or make modifications:

### Prerequisites
- **Go**: Version 1.26 or higher.
- **Git**: To clone the repository.
- **GCC compiler** (Optional, only if using a CGO SQLite driver. Note: LayerInvoice utilizes a modern pure-Go SQLite driver, making it 100% C-Go free and fully cross-compilable!).

### 🚀 Commands

1. **Clone the repository**:
   ```bash
   git clone https://github.com/BasilSaeed/LayerInvoice.git
   cd LayerInvoice
   ```
2. **Setup Configuration**:
   Copy the example environment template:
   ```bash
   cp .env.example .env
   ```
   Modify `.env` to set your desired port and SQLite filename.
3. **Run in Development Mode**:
   ```bash
   go run main.go
   ```
   *Note: When `APP_ENV=development` is enabled in your configuration, the server automatically resets the database files (`layerinvoice.db`, `-wal`, `-shm`) on every boot to guarantee a clean migration slate.*
4. **Compile production binary**:
   ```bash
   go build -ldflags="-s -w" -o layerinvoice .
   ```

---

## 🛡️ License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
