/**
 * User roles in the system
 */
export enum UserRole {
  ADMIN = 'ADMIN',
  USER = 'USER',
}

/**
 * User permissions in the system
 */
export enum UserPermission {
  CREATE_TRACK = 'CREATE_TRACK',
  READ_TRACK = 'READ_TRACK',
  UPDATE_TRACK = 'UPDATE_TRACK',
  DELETE_TRACK = 'DELETE_TRACK',
  MANAGE_USERS = 'MANAGE_USERS',
  MANAGE_API_KEYS = 'MANAGE_API_KEYS',
  ENRICH_METADATA = 'ENRICH_METADATA',
  EXPORT_DDEX = 'EXPORT_DDEX',
}

/**
 * User session information
 */
export interface Session {
  id: string;
  userId: string;
  expiresAt: string;
  createdAt: string;
  lastSeenAt: string;
  userAgent?: string;
  ip?: string;
}

/**
 * User profile information
 */
export interface User {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  permissions: UserPermission[];
  company?: string;
  apiKey?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * Authentication tokens
 */
export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
}

/**
 * Login request payload
 */
export interface LoginRequest {
  email: string;
  password: string;
}

/**
 * Registration request payload
 */
export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
  company?: string;
}

/**
 * Password reset request payload
 */
export interface PasswordResetRequest {
  email: string;
}

/**
 * New password setup payload
 */
export interface NewPasswordRequest {
  token: string;
  password: string;
}

/**
 * Authentication error types
 */
export enum AuthErrorType {
  INVALID_CREDENTIALS = 'INVALID_CREDENTIALS',
  EMAIL_EXISTS = 'EMAIL_EXISTS',
  SERVER_ERROR = 'SERVER_ERROR',
  NETWORK_ERROR = 'NETWORK_ERROR',
  UNAUTHORIZED = 'UNAUTHORIZED',
  FORBIDDEN = 'FORBIDDEN',
}

/**
 * Authentication error with type and message
 */
export interface AuthError {
  type: AuthErrorType;
  message: string;
}

/**
 * Authentication state interface
 */
export interface AuthState {
  user: User | null;
  session: Session | null;
  tokens: AuthTokens | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: AuthError | null;
} 