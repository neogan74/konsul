# ADR-0009: React-Based Admin UI with Vite and Tailwind CSS

**Date**: 2025-10-06

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: frontend, ui, web, admin, tooling

## Context

Konsul provides powerful service discovery and KV store capabilities via REST APIs and CLI tools. However, for operational visibility and ease of use, we need a web-based admin dashboard that allows:

- Real-time monitoring of registered services
- Service health status visualization
- KV store browser and editor
- Service registration/deregistration
- Configuration management
- System health metrics display
- User-friendly alternative to CLI for common operations

### Requirements

1. **Modern UI/UX**: Clean, responsive interface
2. **Real-time updates**: Live service status changes
3. **Performance**: Fast initial load and interactions
4. **Developer Experience**: Easy to develop and maintain
5. **Build tooling**: Fast dev server and optimized production builds
6. **Type safety**: Reduce runtime errors (future TypeScript)
7. **Styling**: Consistent, maintainable design system
8. **Bundle size**: Minimal JavaScript payload
9. **Integration**: Serve from Konsul binary (embedded)

## Decision

We will implement the admin UI using:

1. **React 19** - UI library for component-based architecture
2. **Vite** - Build tool and dev server
3. **Tailwind CSS v4** - Utility-first CSS framework
4. **SPA Architecture** - Single Page Application
5. **Static Build** - Compiled to `web/admin/dist` and served by Fiber

### Technology Stack

**Frontend Framework: React 19**
- Component-based architecture
- Latest features (React Server Components ready)
- Mature ecosystem
- Excellent performance with concurrent rendering

**Build Tool: Vite**
- Lightning-fast HMR (Hot Module Replacement)
- Optimized production builds
- Native ESM support
- Rollup-based bundling

**Styling: Tailwind CSS v4**
- Utility-first approach
- Minimal CSS bundle
- Dark mode ready
- Consistent design tokens
- No runtime CSS-in-JS overhead

**Architecture: SPA (Single Page Application)**
- Client-side routing (React Router future)
- API communication with Konsul backend
- Embedded static build served by Fiber

### Build Output

- **JavaScript**: ~332KB (includes React 19, routing, state management)
- **CSS**: ~20KB (Tailwind utilities)
- **Production**: Optimized, minified, tree-shaken

## Alternatives Considered

### Alternative 1: Server-Side Rendered (SSR) with Next.js
- **Pros**:
  - SEO-friendly (not needed for admin dashboard)
  - Initial page load performance
  - Built-in routing
  - Modern React patterns
- **Cons**:
  - Requires Node.js runtime alongside Go binary
  - More complex deployment
  - Overhead for admin tool use case
  - Harder to embed in Go binary
- **Reason for rejection**: SSR benefits not needed; embedding complexity

### Alternative 2: Vue.js with Vite
- **Pros**:
  - Simpler learning curve
  - Excellent Vite integration
  - Smaller bundle size
  - Great documentation
- **Cons**:
  - Smaller ecosystem than React
  - Less corporate backing
  - Team less familiar
  - Fewer UI component libraries
- **Reason for rejection**: React's ecosystem and team familiarity more valuable

### Alternative 3: Svelte with SvelteKit
- **Pros**:
  - Compile-time framework (no runtime)
  - Smallest bundle size
  - Excellent performance
  - Built-in reactivity
- **Cons**:
  - Smaller ecosystem
  - Less mature tooling
  - Fewer developers know it
  - Limited component libraries
- **Reason for rejection**: Ecosystem maturity and hiring considerations

### Alternative 4: Vanilla JavaScript + Web Components
- **Pros**:
  - No framework dependencies
  - Smallest possible bundle
  - Native browser features
  - Full control
- **Cons**:
  - More boilerplate code
  - Slower development velocity
  - Manual state management
  - Less tooling support
- **Reason for rejection**: Development speed and maintainability concerns

### Alternative 5: HTMX + Go Templates
- **Pros**:
  - Minimal JavaScript
  - Server-rendered HTML
  - Simple architecture
  - Go-native approach
- **Cons**:
  - Limited interactivity
  - Full page reloads
  - Less modern UX
  - Harder real-time updates
- **Reason for rejection**: User experience requirements favor SPA

## Styling: Why Tailwind CSS v4

### Tailwind Alternatives Considered

**CSS Modules**
- **Reason for rejection**: More boilerplate, harder to maintain consistency

**Styled Components / Emotion**
- **Reason for rejection**: Runtime overhead, larger bundles, Flash of Unstyled Content

**Bootstrap / Material UI**
- **Reason for rejection**: Heavy frameworks, harder customization, not Konsul's aesthetic

**Plain CSS / SCSS**
- **Reason for rejection**: Harder to maintain consistency, more files to manage

**Tailwind v4** chosen because:
- Zero runtime overhead (compile-time)
- Minimal CSS output (~20KB)
- Consistent design system via theme
- Fast iteration with utility classes
- Great DX with IntelliSense

## Consequences

### Positive
- **Modern UX**: Responsive, intuitive admin interface
- **Fast development**: Vite HMR and React DevTools accelerate iteration
- **Small bundle**: Optimized builds keep payload minimal
- **Type safety ready**: Easy to add TypeScript later
- **Maintainability**: Component architecture keeps code organized
- **Embeddable**: Static build can be embedded in Go binary
- **No runtime dependencies**: Pure static assets served by Fiber
- **Consistent styling**: Tailwind design tokens ensure consistency
- **Developer experience**: Hot reload, excellent tooling

### Negative
- **JavaScript required**: Users need JS enabled (reasonable for admin dashboard)
- **Build step**: Need Node.js for development (npm/pnpm)
- **Bundle size**: 332KB JS (larger than HTMX approach)
- **SPA complexity**: Client-side routing, state management needed
- **SEO not possible**: Not relevant for admin dashboard
- **Initial load**: Slightly slower than server-rendered HTML
- **Dependency management**: npm packages to maintain

### Neutral
- React ecosystem constantly evolving
- Need to keep dependencies updated
- Build process adds complexity
- Testing requires React Testing Library setup

## Implementation Notes

### Project Structure
```
web/admin/
├── dist/              # Production build output
│   ├── index.html
│   └── assets/
│       ├── index-[hash].js   (~332KB)
│       └── index-[hash].css  (~20KB)
├── src/               # Source code (future)
│   ├── components/
│   ├── pages/
│   ├── hooks/
│   ├── utils/
│   └── App.tsx
├── package.json
├── vite.config.ts
└── tailwind.config.js
```

### Integration with Konsul Backend

**Serving Static Files**:
```go
// Serve admin UI
app.Static("/admin", "./web/admin/dist")
```

**API Communication**:
- UI calls Konsul REST API (`/kv/*`, `/services/*`, etc.)
- JWT/API key authentication
- CORS configuration for development

### Features to Implement

**Dashboard**:
- Service count, health overview
- Recent registrations/deregistrations
- System metrics (CPU, memory, goroutines)
- Quick actions

**Services View**:
- List all registered services
- Filter by name, health status
- View service details (address, port, metadata, TTL)
- Register/deregister services
- Update heartbeat
- Real-time status updates (WebSocket/SSE future)

**KV Store Browser**:
- List all keys with search/filter
- CRUD operations (Create, Read, Update, Delete)
- JSON viewer for complex values
- Bulk operations
- Import/export

**Settings**:
- Server configuration display
- Authentication status
- Connection settings
- Theme toggle (dark/light mode)

### Build Commands

```bash
# Development
cd web/admin && npm run dev

# Production build
cd web/admin && npm run build

# Preview production build
cd web/admin && npm run preview
```

### Future Enhancements
- Add TypeScript for type safety
- WebSocket support for real-time updates
- Service dependency graph visualization
- Metrics dashboards (integrate with Prometheus)
- Health check history timeline
- Dark mode toggle
- Accessibility improvements (ARIA labels, keyboard nav)
- Internationalization (i18n)
- Mobile-responsive optimizations
- Progressive Web App (PWA) features
- Testing suite (Vitest + React Testing Library)

### Performance Considerations
- Code splitting for routes (React.lazy)
- Virtual scrolling for long service lists
- Debounced search inputs
- Optimistic UI updates
- Service worker for offline support
- CDN deployment for assets

### Security Considerations
- CSP headers for XSS protection
- Authentication required for sensitive operations
- CSRF token for state-changing operations
- Input validation and sanitization
- Rate limiting on API calls

## References

- [React 19 Documentation](https://react.dev/)
- [Vite Documentation](https://vitejs.dev/)
- [Tailwind CSS v4](https://tailwindcss.com/)
- [React Router](https://reactrouter.com/)
- [SPA Best Practices](https://web.dev/articles/rendering-on-the-web)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-10-06 | Konsul Team | Initial version |
