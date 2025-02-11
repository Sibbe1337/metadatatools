import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { AuthState, AuthTokens, LoginRequest, RegisterRequest, Session } from '../types/auth';
import { authService } from '../services/authService';
import { useNotifications } from './UIContext';

interface AuthContextType extends AuthState {
  login: (data: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  refreshSession: () => Promise<void>;
  getSessions: () => Promise<Session[]>;
  revokeSession: (sessionId: string) => Promise<void>;
  revokeAllSessions: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

const TOKEN_STORAGE_KEY = 'auth_tokens';
const TOKEN_REFRESH_INTERVAL = 4 * 60 * 1000; // 4 minutes

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const navigate = useNavigate();
  const { showError } = useNotifications();
  const [state, setState] = useState<AuthState>({
    user: null,
    session: null,
    tokens: null,
    isAuthenticated: false,
    isLoading: true,
    error: null,
  });

  // Load tokens from storage
  useEffect(() => {
    const storedTokens = localStorage.getItem(TOKEN_STORAGE_KEY);
    if (storedTokens) {
      const tokens: AuthTokens = JSON.parse(storedTokens);
      setState(prev => ({ ...prev, tokens }));
    }
    setState(prev => ({ ...prev, isLoading: false }));
  }, []);

  // Setup token refresh interval
  useEffect(() => {
    if (state.isAuthenticated) {
      const intervalId = setInterval(() => {
        refreshSession();
      }, TOKEN_REFRESH_INTERVAL);

      return () => clearInterval(intervalId);
    }
  }, [state.isAuthenticated]);

  // Fetch user data when tokens change
  useEffect(() => {
    if (state.tokens && !state.user) {
      loadUser();
    }
  }, [state.tokens]);

  const loadUser = async () => {
    try {
      const user = await authService.getCurrentUser();
      setState(prev => ({
        ...prev,
        user,
        isAuthenticated: true,
        error: null,
      }));
    } catch (error) {
      handleAuthError(error);
    }
  };

  const login = async (data: LoginRequest) => {
    try {
      setState(prev => ({ ...prev, isLoading: true }));
      const { user, session, tokens } = await authService.login(data);
      
      localStorage.setItem(TOKEN_STORAGE_KEY, JSON.stringify(tokens));
      setState(prev => ({
        ...prev,
        user,
        session,
        tokens,
        isAuthenticated: true,
        error: null,
      }));

      navigate('/dashboard');
    } catch (error) {
      handleAuthError(error);
    } finally {
      setState(prev => ({ ...prev, isLoading: false }));
    }
  };

  const register = async (data: RegisterRequest) => {
    try {
      setState(prev => ({ ...prev, isLoading: true }));
      const user = await authService.register(data);
      setState(prev => ({
        ...prev,
        user,
        error: null,
      }));

      // Redirect to login
      navigate('/auth/login');
    } catch (error) {
      handleAuthError(error);
    } finally {
      setState(prev => ({ ...prev, isLoading: false }));
    }
  };

  const logout = async () => {
    try {
      await authService.logout();
      clearAuth();
      navigate('/auth/login');
    } catch (error) {
      handleAuthError(error);
    }
  };

  const refreshSession = async () => {
    try {
      if (!state.tokens?.refreshToken) return;

      const newTokens = await authService.refreshTokens(state.tokens.refreshToken);
      localStorage.setItem(TOKEN_STORAGE_KEY, JSON.stringify(newTokens));
      setState(prev => ({ ...prev, tokens: newTokens }));
    } catch (error) {
      handleAuthError(error);
    }
  };

  const getSessions = async () => {
    try {
      return await authService.getSessions();
    } catch (error) {
      handleAuthError(error);
      return [];
    }
  };

  const revokeSession = async (sessionId: string) => {
    try {
      await authService.revokeSession(sessionId);
    } catch (error) {
      handleAuthError(error);
    }
  };

  const revokeAllSessions = async () => {
    try {
      await authService.revokeAllSessions();
    } catch (error) {
      handleAuthError(error);
    }
  };

  const clearAuth = useCallback(() => {
    localStorage.removeItem(TOKEN_STORAGE_KEY);
    setState({
      user: null,
      session: null,
      tokens: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,
    });
  }, []);

  const handleAuthError = (error: any) => {
    setState(prev => ({
      ...prev,
      error: error,
      isAuthenticated: false,
    }));
    showError(error.message);

    if (error.type === 'SESSION_EXPIRED' || error.type === 'INVALID_TOKEN') {
      clearAuth();
      navigate('/auth/login');
    }
  };

  const value = {
    ...state,
    login,
    register,
    logout,
    refreshSession,
    getSessions,
    revokeSession,
    revokeAllSessions,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}; 