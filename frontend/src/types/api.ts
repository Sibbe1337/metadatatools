import { z } from 'zod';

// API Error Details
export const apiErrorDetailsSchema = z.object({
  field: z.string(),
  message: z.string(),
  code: z.string(),
});

export type ApiErrorDetails = z.infer<typeof apiErrorDetailsSchema>;

// API Error
export class ApiError extends Error {
  constructor(
    message: string,
    public statusCode: number,
    public code?: string,
    public details?: ApiErrorDetails[]
  ) {
    super(message);
    this.name = 'ApiError';
  }

  isValidationError(): boolean {
    return this.statusCode === 400;
  }

  isAuthenticationError(): boolean {
    return this.statusCode === 401 || this.statusCode === 403;
  }

  getFieldError(field: string): string | undefined {
    return this.details?.find((detail) => detail.field === field)?.message;
  }
}

// Type guard for ApiError
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

// API Response Metadata
export const paginationMetaSchema = z.object({
  currentPage: z.number(),
  totalPages: z.number(),
  totalItems: z.number(),
  itemsPerPage: z.number(),
});

export type PaginationMeta = z.infer<typeof paginationMetaSchema>;

// API Response
export const apiResponseSchema = <T extends z.ZodType>(dataSchema: T) =>
  z.object({
    data: dataSchema,
    meta: paginationMetaSchema.optional(),
  });

export type ApiResponse<T> = {
  data: T;
  meta?: PaginationMeta;
}; 