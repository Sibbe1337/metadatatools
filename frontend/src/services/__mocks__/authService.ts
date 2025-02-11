import { AuthError, AuthErrorType, AuthTokens, LoginRequest, User } from '../../types/auth';

export const authService = {
  login: jest.fn(),
  getCurrentUser: jest.fn(),
  register: jest.fn(),
  requestPasswordReset: jest.fn(),
  resetPassword: jest.fn(),
  refreshTokens: jest.fn(),
  logout: jest.fn(),
  getSessions: jest.fn(),
  revokeSession: jest.fn(),
  revokeAllSessions: jest.fn(),
  handleError: jest.fn((error: any): AuthError => ({
    type: AuthErrorType.UNKNOWN,
    message: 'An unexpected error occurred',
  })),
}; 