import { ApiClient } from './client';
import { IApiResponse } from './types';
import {
  Track,
  TrackMetadata,
  ICreateTrackDTO,
  IUpdateTrackDTO,
  ITrackFilters,
} from '../types/track.types';

/**
 * Service for handling track-related API requests
 */
export class TracksApiService {
  private readonly client: ApiClient;
  private readonly baseUrl = '/api/v1/tracks';

  constructor(client: ApiClient) {
    this.client = client;
  }

  /**
   * Fetches a list of tracks with optional filtering
   */
  public async list(filters?: ITrackFilters): Promise<IApiResponse<Track[]>> {
    const queryParams = new URLSearchParams();
    
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) {
          queryParams.append(key, String(value));
        }
      });
    }

    const url = `${this.baseUrl}?${queryParams.toString()}`;
    return this.client.get<Track[]>(url);
  }

  /**
   * Fetches a single track by ID
   */
  public async getById(id: string): Promise<IApiResponse<Track>> {
    return this.client.get<Track>(`${this.baseUrl}/${id}`);
  }

  /**
   * Creates a new track
   */
  public async create(data: ICreateTrackDTO): Promise<IApiResponse<Track>> {
    const formData = new FormData();
    formData.append('file', data.file);
    
    if (data.initialMetadata) {
      formData.append('metadata', JSON.stringify(data.initialMetadata));
    }

    return this.client.uploadFile<Track>(this.baseUrl, data.file);
  }

  /**
   * Updates an existing track's metadata
   */
  public async update(data: IUpdateTrackDTO): Promise<IApiResponse<Track>> {
    return this.client.patch<Track>(`${this.baseUrl}/${data.id}`, {
      metadata: data.metadata,
    });
  }

  /**
   * Deletes a track
   */
  public async delete(id: string): Promise<IApiResponse<void>> {
    return this.client.delete<void>(`${this.baseUrl}/${id}`);
  }

  /**
   * Exports tracks in specified format
   */
  public async export(
    ids: string[],
    format: 'json' | 'ddex'
  ): Promise<IApiResponse<Blob>> {
    return this.client.post<Blob>(
      `${this.baseUrl}/export`,
      { ids, format },
      { responseType: 'blob' }
    );
  }

  /**
   * Validates track metadata using AI
   */
  public async validateMetadata(
    id: string,
    metadata: Partial<TrackMetadata>
  ): Promise<IApiResponse<{ confidence: number; issues?: string[] }>> {
    return this.client.post<{ confidence: number; issues?: string[] }>(
      `${this.baseUrl}/${id}/validate`,
      { metadata }
    );
  }

  /**
   * Enriches track metadata using AI
   */
  public async enrichMetadata(id: string): Promise<IApiResponse<Track>> {
    return this.client.post<Track>(`${this.baseUrl}/${id}/enrich`);
  }
} 