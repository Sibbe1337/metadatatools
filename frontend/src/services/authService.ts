import api from '../config/api';
import {
  AuthError,
  AuthErrorType,
  AuthTokens,
  LoginRequest,
  NewPasswordRequest,
  PasswordResetRequest,
  RegisterRequest,
  Session,
  User,
} from '../types/auth';

const API_BASE_URL = process.env.VITE_API_BASE_URL || 'http://localhost:3000';
const AUTH_API = `${API_BASE_URL}/api/v1/auth`;

export interface LoginCredentials {
  email: string;
  password: string;
}

/**
 * Handles authentication API calls and error mapping
 */
class AuthService {
  private async handleAuthResponse(response: any): Promise<void> {
    const { accessToken, refreshToken } = response.data;
    localStorage.setItem('auth_tokens', JSON.stringify({ accessToken, refreshToken }));
  }

  /**
   * Login with email and password
   */
  async login(data: LoginRequest): Promise<{
    user: User;
    session: Session;
    tokens: AuthTokens;
  }> {
    try {
      const response = await api.post('/auth/login', data);
      const { tokens, user, session } = response.data;
      localStorage.setItem('auth_tokens', JSON.stringify(tokens));
      return { tokens, user, session };
    } catch (error: any) {
      throw this.mapError(error);
    }
  }

  /**
   * Register new user
   */
  async register(data: RegisterRequest): Promise<User> {
    try {
      const response = await api.post('/auth/register', data);
      return response.data;
    } catch (error: any) {
      throw this.mapError(error);
    }
  }

  /**
   * Request password reset
   */
  async requestPasswordReset(data: PasswordResetRequest): Promise<void> {
    try {
      await api.post('/auth/password/reset', data);
    } catch (error: any) {
      throw this.mapError(error);
    }
  }

  /**
   * Set new password with reset token
   */
  async resetPassword(data: NewPasswordRequest): Promise<void> {
    try {
      await api.post('/auth/password/new', data);
    } catch (error: any) {
      throw this.mapError(error);
    }
  }

  /**
   * Refresh authentication tokens
   */
  async refreshTokens(refreshToken: string): Promise<AuthTokens> {
    try {
      const response = await api.post<AuthTokens>('/auth/refresh', { refreshToken });
      return response.data;
    } catch (error: any) {
      throw this.mapError(error);
    }
  }

  /**
   * Logout user
   */
  async logout(): Promise<void> {
    try {
      await api.post('/auth/logout');
    } finally {
      localStorage.removeItem('auth_tokens');
    }
  }

  /**
   * Get current user profile
   */
  async getCurrentUser(): Promise<User | null> {
    try {
      const response = await api.get<User>('/auth/me');
      return response.data;
    } catch (error) {
      return null;
    }
  }

  /**
   * Get user's active sessions
   */
  async getSessions(): Promise<Session[]> {
    try {
      const response = await api.get<Session[]>('/auth/sessions');
      return response.data;
    } catch (error: any) {
      throw this.mapError(error);
    }
  }

  /**
   * Revoke a specific session
   */
  async revokeSession(sessionId: string): Promise<void> {
    try {
      await api.delete(`/auth/sessions/${sessionId}`);
    } catch (error: any) {
      throw this.mapError(error);
    }
  }

  /**
   * Revoke all sessions except current
   */
  async revokeAllSessions(): Promise<void> {
    try {
      await api.delete('/auth/sessions');
    } catch (error: any) {
      throw this.mapError(error);
    }
  }

  /**
   * Map API errors to AuthError type
   */
  private mapError(error: any): AuthError {
    const status = error.response?.status;
    const message = error.response?.data?.error || error.message;

    switch (status) {
      case 401:
        return {
          type: AuthErrorType.INVALID_CREDENTIALS,
          message: 'Invalid email or password',
        };
      case 403:
        return {
          type: AuthErrorType.FORBIDDEN,
          message: 'Access denied',
        };
      case 409:
        return {
          type: AuthErrorType.EMAIL_EXISTS,
          message: 'Email already exists',
        };
      default:
        return {
          type: AuthErrorType.SERVER_ERROR,
          message: message || 'An unexpected error occurred',
        };
    }
  }

  isAuthenticated(): boolean {
    return !!localStorage.getItem('auth_tokens');
  }
}

export const authService = new AuthService(); 