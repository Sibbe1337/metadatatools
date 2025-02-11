import { z } from 'zod';

// Track Status
export const trackStatusSchema = z.enum(['processing', 'ready', 'error']);
export type TrackStatus = z.infer<typeof trackStatusSchema>;

// Track Metadata
export const trackMetadataSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  artist: z.string().min(1, 'Artist is required'),
  genre: z.string().min(1, 'Genre is required'),
  bpm: z.number().optional(),
  key: z.string().optional(),
  mood: z.string().optional(),
  isrc: z.string().optional(),
  tags: z.array(z.string()).default([]),
});

export type TrackMetadata = z.infer<typeof trackMetadataSchema>;

// AI Metadata
export const aiMetadataSchema = z.object({
  confidence: z.number().min(0).max(1),
  needsReview: z.boolean(),
  reviewReason: z.string().optional(),
  model: z.string(),
  processedAt: z.string().datetime(),
});

export type AIMetadata = z.infer<typeof aiMetadataSchema>;

// Track
export const trackSchema = z.object({
  id: z.string().uuid(),
  status: trackStatusSchema,
  metadata: trackMetadataSchema,
  aiMetadata: aiMetadataSchema.optional(),
  createdAt: z.string().datetime(),
  updatedAt: z.string().datetime(),
});

export type Track = z.infer<typeof trackSchema>;

// Track Creation
export const createTrackSchema = z.object({
  file: z.instanceof(File),
  metadata: trackMetadataSchema.partial(),
});

export type CreateTrackDTO = z.infer<typeof createTrackSchema>;

// Track Update
export const updateTrackSchema = z.object({
  id: z.string().uuid(),
  metadata: trackMetadataSchema,
});

export type UpdateTrackDTO = z.infer<typeof updateTrackSchema>;

// Track Filters
export const trackFiltersSchema = z.object({
  search: z.string().optional(),
  genre: z.string().optional(),
  status: trackStatusSchema.optional(),
  limit: z.number().min(1).max(100).optional(),
  offset: z.number().min(0).optional(),
});

export type TrackFilters = z.infer<typeof trackFiltersSchema>; 