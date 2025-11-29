# Admin UI Authentication Implementation

## Overview

The Konsul Admin UI now includes a complete authentication system with JWT-based login, API key management, and protected routes.

## Features Implemented

### 1. Authentication Context (`src/contexts/AuthContext.tsx`)
- **JWT Token Management**: Store and manage access/refresh tokens
- **Local Storage Persistence**: Tokens and user data persist across sessions
- **Automatic Token Refresh**: Seamless token renewal on expiry
- **User State Management**: Track authenticated user with roles and policies

### 2. Login Page (`src/pages/Login.tsx`)
- **Modern UI**: Beautiful dark theme with Tailwind CSS
- **JWT Authentication**: Login with username, user ID, roles, and policies
- **Form Validation**: Required field validation
- **Error Handling**: Clear error messages on login failure
- **Responsive Design**: Works on all device sizes

### 3. API Key Management (`src/pages/APIKeys.tsx`)
- **Create API Keys**: Generate new keys with permissions and expiration
- **List Keys**: View all API keys with status (active/revoked)
- **Revoke Keys**: Disable keys without deletion
- **Delete Keys**: Permanently remove keys
- **Copy to Clipboard**: One-click copy for newly created keys
- **Secure Display**: Show full key only once during creation

### 4. Protected Routes (`src/components/ProtectedRoute.tsx`)
- **Route Protection**: Redirect to login if not authenticated
- **Loading States**: Show spinner while checking auth status
- **Auto-redirect**: Navigate to dashboard after successful login

### 5. User Menu (`src/components/Navbar.tsx`)
- **User Display**: Show username and roles in navbar
- **Dropdown Menu**: Click to view user details
- **Logout Button**: Sign out and clear all auth data
- **Role Badges**: Visual display of user roles

### 6. API Client Integration (`src/lib/api.ts`)
- **Request Interceptor**: Automatically add auth token to all requests
- **Response Interceptor**: Handle 401 errors and refresh expired tokens
- **Auto-retry**: Retry failed requests after token refresh
- **Redirect on Failure**: Navigate to login if refresh fails

## API Endpoints Used

### Authentication
- `POST /auth/login` - Login and get JWT tokens
- `POST /auth/refresh` - Refresh expired access token
- `GET /auth/verify` - Verify current token (unused but available)

### API Key Management
- `POST /auth/apikeys` - Create new API key
- `GET /auth/apikeys` - List all API keys
- `GET /auth/apikeys/:id` - Get specific API key
- `PUT /auth/apikeys/:id` - Update API key
- `DELETE /auth/apikeys/:id` - Delete API key
- `POST /auth/apikeys/:id/revoke` - Revoke API key

## Configuration

### Environment Variables
No additional environment variables needed for the UI. The API base URL is auto-detected from the current hostname.

### Server Configuration
To enable authentication on the Konsul server:

```bash
# Enable authentication
export KONSUL_AUTH_ENABLED=true

# Set JWT secret (required)
export KONSUL_JWT_SECRET="your-super-secret-key-min-32-chars"

# Optional: Customize token expiry
export KONSUL_JWT_EXPIRY=15m
export KONSUL_REFRESH_EXPIRY=168h  # 7 days

# Optional: Require auth for all endpoints
export KONSUL_REQUIRE_AUTH=true

# Start Konsul
./konsul
```

## Usage

### 1. Access the Login Page
Navigate to: `http://localhost:8888/admin/login`

### 2. Login
- **Username**: Your username (e.g., "admin")
- **User ID**: Optional, defaults to username
- **Roles**: Comma-separated roles (e.g., "admin,developer")
- **Policies**: Optional, comma-separated ACL policies

### 3. After Login
- Redirected to dashboard
- User menu appears in top-right
- All API requests include auth token
- Token automatically refreshes before expiry

### 4. Manage API Keys
- Navigate to "API Keys" in sidebar
- Click "Create API Key"
- Copy the key immediately (shown only once!)
- Use key in API requests:
  ```bash
  curl -H "X-API-Key: konsul_abc123..." http://localhost:8888/services/
  ```

### 5. Logout
- Click user menu in navbar
- Click "Sign Out"
- Redirected to login page
- All tokens cleared from localStorage

## Security Features

### Token Storage
- **localStorage**: Tokens stored client-side for persistence
- **Secure Transmission**: Always use HTTPS in production
- **Auto-expiry**: Tokens expire based on server configuration

### Protected Routes
- All main routes require authentication
- Public routes: `/login` only
- Automatic redirect to login if not authenticated

### Token Refresh
- Automatic refresh before token expires
- Retry failed requests after refresh
- Clear all data and redirect if refresh fails

### API Key Security
- Keys shown only once during creation
- Revoke instead of delete for audit trail
- Permissions and expiration for each key

## Development Notes

### Testing Without Auth
If authentication is disabled on the server (`KONSUL_AUTH_ENABLED=false`), the UI will still work for non-auth endpoints. However, the login page and API key management will fail.

### Mock Development
For local development without a backend:
1. Comment out auth checks in `ProtectedRoute.tsx`
2. Disable axios interceptors in `api.ts`
3. Mock API responses in components

### TypeScript
All components are fully typed with TypeScript strict mode enabled.

## Build Information

- **Built**: November 29, 2025
- **Bundle Size**: 358KB JS, 20KB CSS (gzipped: 108KB JS, 4.7KB CSS)
- **React Version**: 19.1.1
- **Vite Version**: 7.1.9
- **TypeScript**: Fully implemented with strict mode

## What's Next?

Remaining UI features to implement:
- [ ] WebSocket real-time updates (replace polling)
- [ ] Settings page with server configuration
- [ ] Light mode and theme toggle
- [ ] Testing suite (Vitest + Playwright)
- [ ] Accessibility improvements (ARIA labels, keyboard nav)