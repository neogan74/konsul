import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import axios from 'axios';

interface User {
  user_id: string;
  username: string;
  roles: string[];
  policies?: string[];
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (username: string, userId: string, roles: string[], policies?: string[]) => Promise<void>;
  logout: () => void;
  refreshAccessToken: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [refreshToken, setRefreshToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Load auth state from localStorage on mount
  useEffect(() => {
    const storedToken = localStorage.getItem('konsul_token');
    const storedRefreshToken = localStorage.getItem('konsul_refresh_token');
    const storedUser = localStorage.getItem('konsul_user');

    if (storedToken && storedUser) {
      setToken(storedToken);
      setRefreshToken(storedRefreshToken);
      setUser(JSON.parse(storedUser));
    }
    setIsLoading(false);
  }, []);

  const login = async (username: string, userId: string, roles: string[], policies: string[] = []) => {
    try {
      const response = await axios.post('/auth/login', {
        user_id: userId,
        username,
        roles,
        policies,
      });

      const { token: accessToken, refresh_token: newRefreshToken } = response.data;
      const userData = { user_id: userId, username, roles, policies };

      setToken(accessToken);
      setRefreshToken(newRefreshToken);
      setUser(userData);

      localStorage.setItem('konsul_token', accessToken);
      localStorage.setItem('konsul_refresh_token', newRefreshToken);
      localStorage.setItem('konsul_user', JSON.stringify(userData));
    } catch (error) {
      console.error('Login failed:', error);
      throw error;
    }
  };

  const logout = () => {
    setToken(null);
    setRefreshToken(null);
    setUser(null);
    localStorage.removeItem('konsul_token');
    localStorage.removeItem('konsul_refresh_token');
    localStorage.removeItem('konsul_user');
  };

  const refreshAccessToken = async () => {
    if (!refreshToken || !user) {
      logout();
      return;
    }

    try {
      const response = await axios.post('/auth/refresh', {
        refresh_token: refreshToken,
        username: user.username,
        roles: user.roles,
        policies: user.policies,
      });

      const { token: newAccessToken, refresh_token: newRefreshToken } = response.data;

      setToken(newAccessToken);
      setRefreshToken(newRefreshToken);

      localStorage.setItem('konsul_token', newAccessToken);
      localStorage.setItem('konsul_refresh_token', newRefreshToken);
    } catch (error) {
      console.error('Token refresh failed:', error);
      logout();
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        refreshToken,
        isAuthenticated: !!token && !!user,
        isLoading,
        login,
        logout,
        refreshAccessToken,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}