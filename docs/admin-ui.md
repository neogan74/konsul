# React Admin UI - Complete Documentation

Comprehensive guide for the Konsul React-based Admin UI.

## Overview

The **Konsul Admin UI** is a React-based web interface that provides a graphical administration console for managing and monitoring your Konsul cluster. It offers an intuitive alternative to the CLI and API for common operations.

### Quick Start

**Access the Admin UI:**
```
http://localhost:8888/admin/
```

**Features:**
- Key-Value store management
- Service discovery visualization
- Health monitoring dashboard
- Metrics and performance graphs
- Backup management
- Configuration editor

---

## Table of Contents

- [Installation](#installation)
- [Architecture](#architecture)
- [Features](#features)
- [Configuration](#configuration)
- [Authentication](#authentication)
- [Components](#components)
- [API Integration](#api-integration)
- [Development](#development)
- [Deployment](#deployment)
- [Customization](#customization)
- [Troubleshooting](#troubleshooting)
- [Security](#security)

---

## Installation

### Pre-built Distribution

The Admin UI comes pre-built with Konsul:

**Location:** `web/admin/dist/`

**Files:**
```
web/admin/dist/
├── index.html          # Main HTML entry point
├── assets/
│   ├── index-*.js      # Bundled JavaScript
│   └── index-*.css     # Bundled CSS
└── vite.svg           # Favicon
```

**No additional installation required** - the UI is served by Konsul automatically.

---

### Serving the UI

**Add static file serving to Konsul:**

```go
// In cmd/konsul/main.go
import "github.com/gofiber/fiber/v2/middleware/filesystem"
import "net/http"

// Serve admin UI static files
app.Use("/admin", filesystem.New(filesystem.Config{
    Root:       http.Dir("./web/admin/dist"),
    Index:      "index.html",
    Browse:     false,
}))

// SPA fallback - serve index.html for all non-API routes
app.Use("/admin/*", func(c *fiber.Ctx) error {
    return c.SendFile("./web/admin/dist/index.html")
})
```

**Access:**
```
http://localhost:8888/admin/
```

---

## Architecture

### Technology Stack

**Frontend:**
- **React** - UI framework
- **Vite** - Build tool and dev server
- **React Router** - Client-side routing
- **Fetch API** - HTTP client for API calls
- **Chart.js / Recharts** - Data visualization (likely)

**Build System:**
- **Vite** - Fast build and HMR
- **TypeScript** - Type safety (if used)
- **ESLint** - Code linting
- **Prettier** - Code formatting

---

### Application Structure

```
web/admin/
├── dist/                    # Built production files
│   ├── index.html
│   └── assets/
├── src/                     # Source files (not included in repo)
│   ├── components/          # React components
│   │   ├── Dashboard/
│   │   ├── KVStore/
│   │   ├── Services/
│   │   ├── Health/
│   │   └── Settings/
│   ├── api/                 # API client
│   ├── hooks/               # Custom React hooks
│   ├── utils/               # Utility functions
│   ├── App.tsx              # Main App component
│   └── main.tsx             # Entry point
├── package.json             # Dependencies
└── vite.config.ts           # Vite configuration
```

---

## Features

### 1. Dashboard

**Overview page showing:**
- System health status
- Active services count
- KV store size
- Request rate metrics
- Recent activity feed
- Quick actions

**Widgets:**
- Service status cards
- Health check results
- Performance graphs
- Alert notifications

---

### 2. Key-Value Store Management

**Capabilities:**
- List all keys with search/filter
- View key details and values
- Create new key-value pairs
- Update existing values
- Delete keys with confirmation
- JSON/text editor with syntax highlighting
- Import/export functionality

**UI Elements:**
```
┌─────────────────────────────────────────────────┐
│  Key-Value Store                        [+ New] │
├─────────────────────────────────────────────────┤
│  🔍 Search: _______________      Filter: [All]  │
├─────────────────────────────────────────────────┤
│  Key                    | Value         | Acti..│
│  app/config/port        | 8080          | [✎][🗑]│
│  app/config/debug       | true          | [✎][🗑]│
│  database/host          | db.example... | [✎][🗑]│
└─────────────────────────────────────────────────┘
```

---

### 3. Service Discovery

**Features:**
- Visual service map/topology
- Service list with health status
- Service details and metadata
- Health check configuration
- Service registration form
- Deregistration with safety checks

**Service Card:**
```
┌──────────────────────────────┐
│ 🟢 web-api                   │
├──────────────────────────────┤
│ Address: 10.0.0.1:8080      │
│ Registered: 2h ago          │
│ Health: Passing (HTTP)      │
│ Checks: 2/2 passing         │
│                              │
│ [Details] [Deregister]      │
└──────────────────────────────┘
```

---

### 4. Health Monitoring

**Dashboard showing:**
- Overall system health
- Service health checks
- Failed health checks
- Check history and trends
- Alert configuration

**Health Check Types:**
- HTTP checks (status visualization)
- TCP checks (connectivity status)
- TTL checks (heartbeat monitoring)
- Custom checks

---

### 5. Metrics & Performance

**Visualizations:**
- Request rate graphs
- Response time percentiles (p50, p95, p99)
- Error rate tracking
- KV operations per second
- Service registration trends
- Rate limit violations

**Time Ranges:**
- Last 5 minutes
- Last hour
- Last 24 hours
- Last 7 days
- Custom range

---

### 6. Backup & Restore

**Functionality:**
- Create backups (one-click)
- List available backups with timestamps
- Download backup files
- Restore from backup with preview
- Schedule automated backups
- Import/export data

**Backup Management:**
```
┌─────────────────────────────────────────────────┐
│  Backups                        [Create Backup] │
├─────────────────────────────────────────────────┤
│  Backup File              | Date      | Size    │
│  backup-20251014-150430  | Today     | 1.2 MB  │
│  backup-20251013-150430  | Yesterday | 1.1 MB  │
│  backup-20251012-150430  | 2 days ago| 1.0 MB  │
│                                                  │
│  [⬇ Download] [↻ Restore] [🗑 Delete]          │
└─────────────────────────────────────────────────┘
```

---

### 7. Configuration Editor

**Settings management:**
- View current configuration
- Edit environment variables (if supported)
- Update log levels dynamically
- Configure rate limits
- Manage authentication settings
- ACL policy editor

---

### 8. Authentication & API Keys

**User management:**
- Login interface
- API key creation
- API key listing and management
- Token refresh
- Session management
- Role-based access control (RBAC) UI

---

## Configuration

### Environment Variables

**Configure UI behavior:**

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_ADMIN_ENABLED` | `true` | Enable admin UI |
| `KONSUL_ADMIN_PATH` | `/admin` | UI base path |
| `KONSUL_API_BASE_URL` | `/` | API base URL |

---

### Build Configuration

**For development builds:**

```bash
cd web/admin
npm install
npm run dev
```

**For production builds:**

```bash
cd web/admin
npm run build
# Output: dist/
```

---

## Authentication

### Login Flow

**1. Navigate to Admin UI:**
```
http://localhost:8888/admin/
```

**2. Login page appears** (if auth enabled):
```
┌────────────────────────┐
│   Konsul Admin         │
│                        │
│   Username: _______    │
│   Password: _______    │
│                        │
│   [Login]              │
└────────────────────────┘
```

**3. Authentication methods:**
- Username/password (JWT)
- API key authentication
- SSO integration (future)

---

### Token Management

**Automatic token handling:**
- JWT tokens stored in localStorage/sessionStorage
- Automatic token refresh before expiry
- Logout clears tokens
- Token included in all API requests

**Example (React code):**
```javascript
// Store token after login
localStorage.setItem('konsul_token', response.token);

// Include in API requests
const headers = {
  'Authorization': `Bearer ${localStorage.getItem('konsul_token')}`
};
```

---

## Components

### Core Components

**1. Dashboard Component:**
```typescript
interface DashboardProps {
  refreshInterval?: number;
}

const Dashboard: React.FC<DashboardProps> = ({ refreshInterval = 30000 }) => {
  // Fetch and display metrics
  // Auto-refresh every 30s
}
```

**2. KVStore Component:**
```typescript
interface KVStoreProps {
  basePath?: string;
}

const KVStore: React.FC<KVStoreProps> = ({ basePath = '/kv' }) => {
  const [keys, setKeys] = useState<string[]>([]);
  const [selectedKey, setSelectedKey] = useState<string | null>(null);

  // CRUD operations
}
```

**3. ServiceList Component:**
```typescript
interface Service {
  name: string;
  address: string;
  port: number;
  health: 'passing' | 'warning' | 'critical';
}

const ServiceList: React.FC = () => {
  const [services, setServices] = useState<Service[]>([]);
  // Display and manage services
}
```

---

### Shared Components

**API Client:**
```typescript
class KonsulAPI {
  private baseURL: string;
  private token: string | null;

  constructor(baseURL: string) {
    this.baseURL = baseURL;
    this.token = localStorage.getItem('konsul_token');
  }

  async getKV(key: string): Promise<any> {
    const response = await fetch(`${this.baseURL}/kv/${key}`, {
      headers: this.getHeaders(),
    });
    return response.json();
  }

  async listServices(): Promise<Service[]> {
    const response = await fetch(`${this.baseURL}/services/`, {
      headers: this.getHeaders(),
    });
    return response.json();
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }
    return headers;
  }
}
```

---

## API Integration

### REST API Client

**Base configuration:**
```typescript
const API_BASE_URL = process.env.VITE_API_BASE_URL || '';

// Create API client instance
const api = new KonsulAPI(API_BASE_URL);
```

---

### API Endpoints Used

**KV Store:**
- `GET /kv/` - List keys
- `GET /kv/:key` - Get value
- `PUT /kv/:key` - Set value
- `DELETE /kv/:key` - Delete key

**Services:**
- `GET /services/` - List services
- `GET /services/:name` - Get service
- `PUT /register` - Register service
- `DELETE /deregister/:name` - Deregister service
- `PUT /heartbeat/:name` - Send heartbeat

**Health:**
- `GET /health` - System health
- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe
- `GET /health/checks` - All health checks
- `GET /health/service/:name` - Service health checks

**Backup:**
- `POST /backup` - Create backup
- `GET /backups` - List backups
- `POST /restore` - Restore backup
- `GET /export` - Export data

**Metrics:**
- `GET /metrics` - Prometheus metrics (parsed for UI)

**Authentication:**
- `POST /auth/login` - Login
- `POST /auth/refresh` - Refresh token
- `GET /auth/verify` - Verify token
- `POST /auth/apikeys` - Create API key
- `GET /auth/apikeys` - List API keys

---

### Error Handling

**Consistent error handling:**
```typescript
async function handleAPICall<T>(
  apiCall: () => Promise<T>
): Promise<T | null> {
  try {
    return await apiCall();
  } catch (error) {
    if (error.response?.status === 401) {
      // Unauthorized - redirect to login
      window.location.href = '/admin/login';
    } else if (error.response?.status === 429) {
      // Rate limited
      showNotification('Too many requests. Please try again later.', 'warning');
    } else {
      showNotification('An error occurred. Please try again.', 'error');
    }
    console.error('API Error:', error);
    return null;
  }
}
```

---

## Development

### Prerequisites

**Required:**
- Node.js 18+
- npm or yarn
- Modern browser (Chrome, Firefox, Safari, Edge)

---

### Development Setup

**1. Install dependencies:**
```bash
cd web/admin
npm install
```

**2. Configure development environment:**
```bash
# Create .env file
cat > .env <<EOF
VITE_API_BASE_URL=http://localhost:8888
VITE_POLLING_INTERVAL=30000
EOF
```

**3. Start development server:**
```bash
npm run dev
# Opens http://localhost:5173
```

**4. Start Konsul backend:**
```bash
# In another terminal
cd ../..
go run cmd/konsul/main.go
```

---

### Project Structure (Development)

**Recommended structure:**
```
web/admin/src/
├── components/
│   ├── Dashboard/
│   │   ├── Dashboard.tsx
│   │   ├── MetricsCard.tsx
│   │   └── ActivityFeed.tsx
│   ├── KVStore/
│   │   ├── KVList.tsx
│   │   ├── KVEditor.tsx
│   │   └── KVForm.tsx
│   ├── Services/
│   │   ├── ServiceList.tsx
│   │   ├── ServiceCard.tsx
│   │   └── ServiceForm.tsx
│   ├── Health/
│   │   ├── HealthDashboard.tsx
│   │   └── CheckStatus.tsx
│   ├── Auth/
│   │   ├── Login.tsx
│   │   └── APIKeyManager.tsx
│   └── Layout/
│       ├── Header.tsx
│       ├── Sidebar.tsx
│       └── Footer.tsx
├── api/
│   ├── client.ts           # API client
│   ├── types.ts            # TypeScript types
│   └── hooks.ts            # Custom hooks for API
├── hooks/
│   ├── usePolling.ts       # Auto-refresh hook
│   ├── useAuth.ts          # Authentication hook
│   └── useNotification.ts  # Notification hook
├── utils/
│   ├── formatters.ts       # Data formatting
│   ├── validators.ts       # Input validation
│   └── constants.ts        # App constants
├── styles/
│   ├── global.css
│   └── themes.css
├── App.tsx
├── main.tsx
└── router.tsx
```

---

### Custom Hooks

**usePolling - Auto-refresh data:**
```typescript
function usePolling<T>(
  fetchFn: () => Promise<T>,
  interval: number = 30000
): { data: T | null; loading: boolean; error: Error | null } {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const fetch = async () => {
      try {
        const result = await fetchFn();
        setData(result);
        setError(null);
      } catch (err) {
        setError(err as Error);
      } finally {
        setLoading(false);
      }
    };

    fetch();
    const timer = setInterval(fetch, interval);
    return () => clearInterval(timer);
  }, [fetchFn, interval]);

  return { data, loading, error };
}
```

**Usage:**
```typescript
const Dashboard = () => {
  const { data: health } = usePolling(() => api.getHealth(), 30000);
  // Auto-refreshes every 30 seconds
};
```

---

## Deployment

### Production Build

**Build for production:**
```bash
cd web/admin
npm run build
# Output: dist/
```

**Optimize build:**
```bash
# In vite.config.ts
export default defineConfig({
  build: {
    minify: 'terser',
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
        },
      },
    },
  },
});
```

---

### Docker Deployment

**Multi-stage Dockerfile:**
```dockerfile
# Stage 1: Build React app
FROM node:18-alpine AS ui-builder
WORKDIR /app/web/admin
COPY web/admin/package*.json ./
RUN npm ci
COPY web/admin/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o konsul ./cmd/konsul

# Stage 3: Production image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=go-builder /app/konsul .
COPY --from=ui-builder /app/web/admin/dist ./web/admin/dist
EXPOSE 8888
CMD ["./konsul"]
```

---

### CDN Deployment

**For separate UI deployment:**
```bash
# Build with absolute API URL
VITE_API_BASE_URL=https://konsul.example.com npm run build

# Deploy to CDN
aws s3 sync dist/ s3://konsul-ui-bucket/ --delete
aws cloudfront create-invalidation --distribution-id XXX --paths "/*"
```

---

## Customization

### Theming

**CSS Variables:**
```css
:root {
  --primary-color: #4CAF50;
  --secondary-color: #2196F3;
  --danger-color: #F44336;
  --success-color: #4CAF50;
  --warning-color: #FF9800;
  --background-color: #f5f5f5;
  --text-color: #333;
}

[data-theme="dark"] {
  --background-color: #1e1e1e;
  --text-color: #e0e0e0;
}
```

---

### Custom Components

**Add custom dashboard widget:**
```typescript
// components/Dashboard/CustomWidget.tsx
export const CustomWidget: React.FC = () => {
  return (
    <div className="widget">
      <h3>Custom Metric</h3>
      {/* Your custom content */}
    </div>
  );
};

// Add to Dashboard.tsx
import { CustomWidget } from './CustomWidget';

const Dashboard = () => (
  <div className="dashboard">
    <CustomWidget />
    {/* Other widgets */}
  </div>
);
```

---

### Configuration

**Runtime configuration:**
```typescript
// config.ts
interface AppConfig {
  apiBaseURL: string;
  pollingInterval: number;
  maxRetries: number;
  features: {
    backup: boolean;
    metrics: boolean;
    acl: boolean;
  };
}

const config: AppConfig = {
  apiBaseURL: import.meta.env.VITE_API_BASE_URL || '',
  pollingInterval: parseInt(import.meta.env.VITE_POLLING_INTERVAL) || 30000,
  maxRetries: 3,
  features: {
    backup: true,
    metrics: true,
    acl: true,
  },
};

export default config;
```

---

## Troubleshooting

### Issue: UI Not Loading

**Symptoms:**
- Blank page
- 404 errors
- Console errors

**Solutions:**
1. **Verify static file serving:**
   ```go
   // Check main.go has filesystem middleware
   app.Use("/admin", filesystem.New(...))
   ```

2. **Check build output:**
   ```bash
   ls -la web/admin/dist/
   # Should contain index.html and assets/
   ```

3. **Rebuild UI:**
   ```bash
   cd web/admin
   npm run build
   ```

---

### Issue: API Calls Failing

**Symptoms:**
- Network errors in console
- "Failed to fetch" errors
- CORS errors

**Solutions:**
1. **Check API base URL:**
   ```typescript
   console.log('API Base URL:', config.apiBaseURL);
   ```

2. **Verify CORS configuration:**
   ```go
   // In main.go
   app.Use(cors.New(cors.Config{
       AllowOrigins: "*",
       AllowHeaders: "Origin, Content-Type, Accept, Authorization",
   }))
   ```

3. **Check network tab:**
   - Open browser DevTools
   - Network tab
   - Verify requests are being sent

---

### Issue: Authentication Not Working

**Symptoms:**
- Login fails
- Redirects to login repeatedly
- Token not persisting

**Solutions:**
1. **Check localStorage:**
   ```javascript
   console.log('Token:', localStorage.getItem('konsul_token'));
   ```

2. **Verify token in requests:**
   ```javascript
   // Check Network tab → Headers
   // Should see: Authorization: Bearer <token>
   ```

3. **Clear storage and re-login:**
   ```javascript
   localStorage.clear();
   sessionStorage.clear();
   ```

---

### Issue: Slow Performance

**Symptoms:**
- UI laggy
- Slow page loads
- High CPU usage

**Solutions:**
1. **Reduce polling interval:**
   ```typescript
   // Increase from 10s to 60s
   const { data } = usePolling(fetchFn, 60000);
   ```

2. **Enable code splitting:**
   ```typescript
   // Use lazy loading
   const Dashboard = lazy(() => import('./components/Dashboard'));
   ```

3. **Optimize bundle:**
   ```bash
   npm run build -- --analyze
   ```

---

## Security

### Best Practices

**1. Never store sensitive data:**
```typescript
// ❌ Bad
localStorage.setItem('password', userPassword);

// ✅ Good
localStorage.setItem('konsul_token', jwtToken);
// Token has expiry and can be revoked
```

**2. Validate all inputs:**
```typescript
function validateKey(key: string): boolean {
  // Prevent path traversal
  return !key.includes('../') && !key.includes('..\\');
}
```

**3. Use HTTPS in production:**
```typescript
if (window.location.protocol !== 'https:' &&
    window.location.hostname !== 'localhost') {
  console.warn('Admin UI should be served over HTTPS');
}
```

**4. Implement CSP headers:**
```go
// In main.go
app.Use(func(c *fiber.Ctx) error {
    c.Set("Content-Security-Policy",
        "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
    return c.Next()
})
```

**5. Rate limit API calls:**
```typescript
// Client-side rate limiting
class RateLimiter {
  private queue: number[] = [];
  private limit: number = 100;
  private window: number = 60000; // 1 minute

  canMakeRequest(): boolean {
    const now = Date.now();
    this.queue = this.queue.filter(t => t > now - this.window);
    return this.queue.length < this.limit;
  }

  recordRequest() {
    this.queue.push(Date.now());
  }
}
```

---

## See Also

- [konsulctl CLI Documentation](konsulctl.md)
- [Authentication Documentation](authentication.md)
- [API Reference](../README.md)
- [React Documentation](https://react.dev/)
- [Vite Documentation](https://vitejs.dev/)

---

## Changelog

- **2025-10-14**: Initial comprehensive documentation
- **Version**: 0.1.0
- **Status**: ⚠️ In Development

---

## Future Enhancements

Planned features for the Admin UI:

- [ ] Real-time updates via WebSocket
- [ ] Advanced search and filtering
- [ ] Service dependency graph visualization
- [ ] Alerting rules configuration
- [ ] User management interface
- [ ] Audit log viewer
- [ ] Query builder for metrics
- [ ] Dark mode toggle
- [ ] Mobile-responsive design
- [ ] Internationalization (i18n)
- [ ] Accessibility improvements (WCAG 2.1)
- [ ] Keyboard shortcuts
- [ ] Export reports (PDF, CSV)
- [ ] Embedded documentation
- [ ] Interactive tutorials
