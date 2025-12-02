# Page snapshot

```yaml
- generic [ref=e4]:
  - generic [ref=e5]:
    - img [ref=e7]
    - heading "Konsul Admin" [level=1] [ref=e10]
    - paragraph [ref=e11]: Sign in to manage your cluster
  - generic [ref=e12]:
    - generic [ref=e13]:
      - generic [ref=e14]:
        - generic [ref=e15]: Username
        - generic [ref=e16]:
          - img
          - textbox "Enter username" [ref=e17]
      - generic [ref=e18]:
        - generic [ref=e19]: User ID (optional)
        - generic [ref=e20]:
          - img
          - textbox "Defaults to username" [ref=e21]
      - generic [ref=e22]:
        - generic [ref=e23]: Roles (comma-separated)
        - generic [ref=e24]:
          - img
          - textbox "admin, developer" [ref=e25]: admin
      - generic [ref=e26]:
        - generic [ref=e27]: Policies (comma-separated, optional)
        - textbox "developer, readonly" [ref=e28]
      - button "Sign In" [ref=e29]
    - paragraph [ref=e31]: This login uses Konsul's JWT authentication system. Ensure authentication is enabled on the server.
  - paragraph [ref=e33]: "Development mode: Authentication may be disabled on the server"
```