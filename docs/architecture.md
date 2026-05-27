### **1. High-Level System Architecture**

This diagram focuses strictly on system boundaries, logical grouping, and the primary flow of data. Implementation details and workflows have been abstracted away to provide a clean 30-second overview of the stack.

```mermaid
flowchart TD
    classDef frontend fill:#3b82f6,color:#fff,stroke:#1d4ed8,stroke-width:2px
    classDef core fill:#8b5cf6,color:#fff,stroke:#6d28d9,stroke-width:2px
    classDef module fill:#10b981,color:#fff,stroke:#047857,stroke-width:2px
    classDef engine fill:#f59e0b,color:#fff,stroke:#d97706,stroke-width:2px
    classDef storage fill:#64748b,color:#fff,stroke:#475569,stroke-width:2px

    Browser[Client UI <br> PicoCSS / Alpine / HTMX]:::frontend
    
    subgraph Core Server [Go Backend]
        Router[Chi Router & Middleware]:::core
        Auth[Session Store]:::core
    end
    
    subgraph Business Modules
        CRM[Clients]:::module
        Billing[Quotes / Invoices / Payments]:::module
        Dash[Dashboard Aggregator]:::module
    end
    
    subgraph Technical Engines
        Calc[3D Cost Calculator]:::engine
        PDF[PDF Renderer]:::engine
        Mail[SMTP Mailer]:::engine
        Workers[Background Workers]:::engine
    end

    DB[(SQLite WAL)]:::storage

    Browser -->|HTTP Requests| Router
    Router -->|Authenticate| Auth
    Router -->|Route to| Business Modules
    
    Business Modules -->|Trigger| Technical Engines
    Business Modules <-->|Read / Write| DB
    Technical Engines -.->|Read / Write| DB
    Auth <-->|Verify| DB

```

---

### **2. Request Lifecycle & Authentication Flow**

Instead of mapping every single router connection to a database node, this sequence diagram explains the general pattern every request follows.

```mermaid
sequenceDiagram
    actor Client
    participant App as App (HTMX/JS)
    participant Router as Chi Router
    participant Session as Session Store
    participant DB as SQLite
    participant Handler as Business Handler

    Client->>App: Interaction
    App->>Router: HTTP Request
    Router->>Session: Validate Middleware
    Session->>DB: Check Token
    DB-->>Session: Token Valid
    Router->>Handler: Forward Request
    Handler->>DB: Execute Query
    DB-->>Handler: Result Set
    Handler-->>App: Render HTML Snippet
    App-->>Client: Update UI

```

---

### **3. Quote to Invoice Conversion**

This abstracts the "1-Click Convert" logic into a dedicated process flow, showing exactly how state is transferred from an estimate to a billable entity.

```mermaid
sequenceDiagram
    actor User
    participant Q as Quotes Module
    participant I as Invoices Module
    participant DB as SQLite

    User->>Q: Click "Convert to Invoice"
    Q->>DB: Fetch Quote Details
    DB-->>Q: Quote Data
    Q->>I: Initialize Draft Invoice
    I->>DB: Duplicate Line Items (Preserve 3D Print Type)
    DB-->>I: Save Confirmed
    I-->>User: Redirect to Invoice Editor

```

---

### **4. 3D Print Cost Calculation Flow**

The cost calculator serves two distinct purposes: generating a live UI preview and officially logging a print job. This sequence diagram clarifies that dual behavior.

```mermaid
sequenceDiagram
    participant UI as Browser (HTMX)
    participant Calc as Cost Engine
    participant DB as SQLite

    UI->>Calc: POST Print Parameters
    Note over Calc: Calculate: Base Cost + Markups <br/> Machine Time + Failure Buffer
    
    alt Live Preview Request
        Calc-->>UI: Render Formatted Cost HTML
    else Save Print Job Request
        Calc->>Calc: Format Multi-line Description
        Calc->>DB: Save Detailed Line Item
        DB-->>Calc: Confirm Insert
        Calc-->>UI: Append Line Item to View
    end

```

---

### **5. PDF Generation & Debounced Mailer Workflow**

Debouncing and background processing are captures of timing mechanisms and background hand-offs.

```mermaid
sequenceDiagram
    participant UI as Browser
    participant Inv as Invoice Module
    participant Debounce as Worker
    participant PDF as Gofpdf (Vector Engine)
    participant SMTP as Mailer Engine
    actor Inbox as Client Email

    UI->>Inv: Save Line Item (Keystroke)
    Inv->>Debounce: Emit Save Event
    Note over Debounce: Wait 5 Seconds<br/>(Drop rapid successive events)
    Debounce->>PDF: Generate Vector PDF
    PDF-->>Debounce: Compiled PDF Bytes
    Debounce->>SMTP: Dispatch Payload
    SMTP->>Inbox: Deliver Email + Attachment

```