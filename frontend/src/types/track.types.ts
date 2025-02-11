import { z } from 'zod';

/**
 * Represents the status of a track in the system
 */
export const TrackStatusEnum = {
  PROCESSING: 'processing',
  READY: 'ready',
  ERROR: 'error',
} as const;

export type TrackStatus = typeof TrackStatusEnum[keyof typeof TrackStatusEnum];

/**
 * Base schema for track metadata validation
 */
export const trackMetadataSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  artist: z.string().min(1, 'Artist is required'),
  genre: z.string().min(1, 'Genre is required'),
  bpm: z.number().min(1, 'BPM must be greater than 0').optional(),
  key: z.string().optional(),
  mood: z.string().optional(),
  isrc: z.string().regex(/^[A-Z]{2}[A-Z0-9]{3}[0-9]{7}$/, 'Invalid ISRC format').optional(),
  tags: z.array(z.string()).optional(),
});

/**
 * Schema for AI-generated metadata
 */
export const aiMetadataSchema = z.object({
  confidence: z.number().min(0).max(1),
  needsReview: z.boolean(),
  reviewReason: z.string().optional(),
  model: z.string(),
  processedAt: z.date(),
});

/**
 * Complete track schema including system fields
 */
export const trackSchema = z.object({
  id: z.string().uuid(),
  status: z.enum([TrackStatusEnum.PROCESSING, TrackStatusEnum.READY, TrackStatusEnum.ERROR]),
  createdAt: z.date(),
  updatedAt: z.date(),
  metadata: trackMetadataSchema,
  aiMetadata: aiMetadataSchema.optional(),
});

// Type definitions derived from schemas
export type TrackMetadata = z.infer<typeof trackMetadataSchema>;
export type AIMetadata = z.infer<typeof aiMetadataSchema>;
export type Track = z.infer<typeof trackSchema>;

/**
 * Interface for track creation
 */
export interface ICreateTrackDTO {
  file: File;
  initialMetadata?: Partial<TrackMetadata>;
}

/**
 * Interface for track update
 */
export interface IUpdateTrackDTO {
  id: string;
  metadata: Partial<TrackMetadata>;
}

/**
 * Interface for track filtering
 */
export interface ITrackFilters {
  search?: string;
  genre?: string;
  status?: TrackStatus;
  needsReview?: boolean;
  page?: number;
  limit?: number;
  sortBy?: keyof Track;
  sortOrder?: 'asc' | 'desc';
} 