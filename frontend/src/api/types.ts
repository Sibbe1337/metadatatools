/**
 * Base interface for API response pagination metadata
 */
export interface IPaginationMeta {
  currentPage: number;
  totalPages: number;
  totalItems: number;
  itemsPerPage: number;
}

/**
 * Generic interface for successful API responses
 */
export interface IApiResponse<T> {
  data: T;
  meta?: IPaginationMeta;
}

/**
 * Interface for API error details
 */
export interface IApiErrorDetails {
  field: string;
  message: string;
  code: string;
}

/**
 * Custom error class for API errors
 */
export class ApiError extends Error {
  public readonly statusCode: number;
  public readonly code: string;
  public readonly details?: IApiErrorDetails[];

  constructor(
    message: string,
    statusCode: number,
    code: string = 'UNKNOWN_ERROR',
    details?: IApiErrorDetails[]
  ) {
    super(message);
    this.name = 'ApiError';
    this.statusCode = statusCode;
    this.code = code;
    this.details = details;
  }

  /**
   * Checks if the error is a validation error
   */
  public isValidationError(): boolean {
    return this.statusCode === 422;
  }

  /**
   * Checks if the error is an authentication error
   */
  public isAuthError(): boolean {
    return this.statusCode === 401 || this.statusCode === 403;
  }

  /**
   * Gets field-specific error message
   */
  public getFieldError(fieldName: string): string | undefined {
    return this.details?.find(detail => detail.field === fieldName)?.message;
  }
}

/**
 * Type guard to check if a response is an error
 */
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

/**
 * HTTP methods supported by the API
 */
export const HttpMethod = {
  GET: 'GET',
  POST: 'POST',
  PUT: 'PUT',
  DELETE: 'DELETE',
  PATCH: 'PATCH',
} as const;

export type HttpMethodType = typeof HttpMethod[keyof typeof HttpMethod]; 