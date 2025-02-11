import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { TrackList } from '../components/organisms/TrackList';
import { Input } from '../components/atoms/Input';
import { Button } from '../components/atoms/Button';
import { useTrackApi } from '../hooks/useTrackApi';
import { useUI } from '../context/UIContext';
import type { Track } from '../types/track.types';

const Tracks = () => {
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedGenre, setSelectedGenre] = useState<string>('all');
  const { showNotification, showModal, hideModal } = useUI();

  const {
    useTracks,
    useDeleteTrack,
    useExportTracks,
  } = useTrackApi();

  // Fetch tracks with filters
  const { data: tracks, isLoading, error } = useTracks({
    search: searchQuery || undefined,
    genre: selectedGenre === 'all' ? undefined : selectedGenre,
  });

  // Delete mutation
  const deleteMutation = useDeleteTrack();

  // Export mutation
  const exportMutation = useExportTracks();

  const genres = ['all', 'House', 'Jazz', 'Rock', 'Hip Hop', 'Classical'];

  const handleEdit = (track: Track) => {
    navigate(`/tracks/${track.id}/edit`);
  };

  const handleDelete = async (track: Track) => {
    const modalId = showModal(
      <div className="space-y-4">
        <h3 className="text-lg font-medium text-gray-900">Delete Track</h3>
        <p className="text-sm text-gray-500">
          Are you sure you want to delete "{track.metadata.title}"? This action cannot be undone.
        </p>
        <div className="flex justify-end space-x-3">
          <Button variant="secondary" onClick={() => hideModal(modalId)}>
            Cancel
          </Button>
          <Button
            variant="danger"
            onClick={async () => {
              try {
                await deleteMutation.mutateAsync(track.id);
                hideModal(modalId);
                showNotification('success', 'Track deleted successfully');
              } catch (err) {
                showNotification('error', 'Failed to delete track');
              }
            }}
            isLoading={deleteMutation.isPending}
          >
            Delete
          </Button>
        </div>
      </div>
    );
  };

  const handleExport = async (track: Track) => {
    try {
      const response = await exportMutation.mutateAsync({
        ids: [track.id],
        format: 'json',
      });

      // Create a download link
      const blob = new Blob([response.data], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${track.metadata.title}.json`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      showNotification('success', 'Track exported successfully');
    } catch (err) {
      showNotification('error', 'Failed to export track');
    }
  };

  const handleExportAll = async () => {
    if (!tracks?.length) return;

    try {
      const response = await exportMutation.mutateAsync({
        ids: tracks.map(track => track.id),
        format: 'json',
      });

      // Create a download link
      const blob = new Blob([response.data], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'tracks.json';
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      showNotification('success', 'Tracks exported successfully');
    } catch (err) {
      showNotification('error', 'Failed to export tracks');
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">Tracks</h1>
        <p className="mt-1 text-sm text-gray-500">
          Manage and enrich your track metadata.
        </p>
      </div>

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex-1 max-w-sm">
          <Input
            placeholder="Search tracks..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <div className="flex items-center gap-2">
          <select
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 sm:text-sm"
            value={selectedGenre}
            onChange={(e) => setSelectedGenre(e.target.value)}
          >
            {genres.map((genre) => (
              <option key={genre} value={genre}>
                {genre.charAt(0).toUpperCase() + genre.slice(1)}
              </option>
            ))}
          </select>
          <Button
            variant="secondary"
            onClick={handleExportAll}
            disabled={!tracks?.length || exportMutation.isPending}
          >
            Export All
          </Button>
          <Button onClick={() => navigate('/upload')}>
            Upload New
          </Button>
        </div>
      </div>

      <TrackList
        tracks={tracks || []}
        isLoading={isLoading}
        error={error?.message}
        onEdit={handleEdit}
        onDelete={handleDelete}
        onExport={handleExport}
      />
    </div>
  );
};

export default Tracks; 