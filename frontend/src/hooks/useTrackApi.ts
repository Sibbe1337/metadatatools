import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useUI } from '../context/UIContext';
import { Track, TrackMetadata } from '../types/track';
import { ApiError } from '../types/api';

interface TrackFilters {
  search?: string;
  genre?: string;
  status?: string;
  limit?: number;
  offset?: number;
}

interface TrackUploadResponse {
  trackId: string;
  uploadUrl: string;
}

export const useTrackApi = () => {
  const queryClient = useQueryClient();
  const { showNotification } = useUI();

  // Fetch tracks with filters
  const useTracks = (filters?: TrackFilters) => {
    return useQuery<Track[], ApiError>({
      queryKey: ['tracks', filters],
      queryFn: async () => {
        const params = new URLSearchParams();
        if (filters?.search) params.append('search', filters.search);
        if (filters?.genre) params.append('genre', filters.genre);
        if (filters?.status) params.append('status', filters.status);
        if (filters?.limit) params.append('limit', filters.limit.toString());
        if (filters?.offset) params.append('offset', filters.offset.toString());

        const response = await fetch(`/api/tracks?${params.toString()}`);
        if (!response.ok) {
          throw new ApiError('Failed to fetch tracks', response.status);
        }
        return response.json();
      },
    });
  };

  // Fetch a single track by ID
  const useTrack = (id: string) => {
    return useQuery<Track, ApiError>({
      queryKey: ['track', id],
      queryFn: async () => {
        const response = await fetch(`/api/tracks/${id}`);
        if (!response.ok) {
          throw new ApiError('Failed to fetch track', response.status);
        }
        return response.json();
      },
      enabled: !!id,
    });
  };

  // Create a new track
  const createTrackMutation = useMutation<TrackUploadResponse, ApiError, File>({
    mutationFn: async (file: File) => {
      // First, create the track record
      const formData = new FormData();
      formData.append('file', file);

      const response = await fetch('/api/tracks', {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        throw new ApiError('Failed to create track', response.status);
      }

      return response.json();
    },
    onSuccess: () => {
      showNotification('success', 'Track created successfully');
      queryClient.invalidateQueries({ queryKey: ['tracks'] });
    },
    onError: (error) => {
      showNotification('error', error.message);
    },
  });

  // Update track metadata
  const updateTrackMutation = useMutation<Track, ApiError, { id: string; metadata: TrackMetadata }>({
    mutationFn: async ({ id, metadata }) => {
      const response = await fetch(`/api/tracks/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ metadata }),
      });

      if (!response.ok) {
        throw new ApiError('Failed to update track', response.status);
      }

      return response.json();
    },
    onSuccess: (data) => {
      showNotification('success', 'Track updated successfully');
      queryClient.invalidateQueries({ queryKey: ['tracks'] });
      queryClient.invalidateQueries({ queryKey: ['track', data.id] });
    },
    onError: (error) => {
      showNotification('error', error.message);
    },
  });

  // Delete a track
  const deleteTrackMutation = useMutation<void, ApiError, string>({
    mutationFn: async (id: string) => {
      const response = await fetch(`/api/tracks/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new ApiError('Failed to delete track', response.status);
      }
    },
    onSuccess: () => {
      showNotification('success', 'Track deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['tracks'] });
    },
    onError: (error) => {
      showNotification('error', error.message);
    },
  });

  // Export tracks
  const exportTracksMutation = useMutation<string, ApiError, void>({
    mutationFn: async () => {
      const response = await fetch('/api/tracks/export', {
        method: 'POST',
      });

      if (!response.ok) {
        throw new ApiError('Failed to export tracks', response.status);
      }

      return response.json();
    },
    onSuccess: (downloadUrl) => {
      showNotification('success', 'Tracks exported successfully');
      window.open(downloadUrl, '_blank');
    },
    onError: (error) => {
      showNotification('error', error.message);
    },
  });

  return {
    useTracks,
    useTrack,
    createTrackMutation,
    updateTrackMutation,
    deleteTrackMutation,
    exportTracksMutation,
  };
}; 