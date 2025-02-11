import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card } from '../components/atoms/Card';
import { TrackUpload } from '../components/molecules/TrackUpload';
import { TrackMetadataForm } from '../components/molecules/TrackMetadataForm';
import { useTrackApi } from '../hooks/useTrackApi';
import { useUI } from '../context/UIContext';
import type { TrackMetadata } from '../types/track.types';

const Upload = () => {
  const navigate = useNavigate();
  const [currentStep, setCurrentStep] = useState<'upload' | 'metadata'>('upload');
  const [trackId, setTrackId] = useState<string | null>(null);
  const { showNotification } = useUI();

  const { useCreateTrack, useUpdateTrack } = useTrackApi();
  const createTrackMutation = useCreateTrack();
  const updateTrackMutation = useUpdateTrack();

  const handleUpload = async (file: File) => {
    try {
      const response = await createTrackMutation.mutateAsync({
        file,
      });
      setTrackId(response.data.id);
      setCurrentStep('metadata');
      showNotification('success', 'Track uploaded successfully');
    } catch (err) {
      showNotification('error', 'Failed to upload track');
    }
  };

  const handleMetadataSubmit = async (data: TrackMetadata) => {
    if (!trackId) return;

    try {
      await updateTrackMutation.mutateAsync({
        id: trackId,
        metadata: data,
      });
      showNotification('success', 'Track metadata saved successfully');
      navigate('/tracks');
    } catch (err) {
      showNotification('error', 'Failed to save track metadata');
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">Upload Track</h1>
        <p className="mt-1 text-sm text-gray-500">
          Upload your track and enrich its metadata using AI.
        </p>
      </div>

      <div className="flex items-center">
        <div className="flex-1 border-t-2 border-gray-200" />
        <nav className="flex items-center space-x-4 px-4" aria-label="Progress">
          <button
            type="button"
            className={`
              flex items-center text-sm font-medium
              ${currentStep === 'upload' ? 'text-primary-600' : 'text-gray-500'}
            `}
            onClick={() => setCurrentStep('upload')}
          >
            <span
              className={`
                flex-shrink-0 w-8 h-8 flex items-center justify-center rounded-full border-2
                ${currentStep === 'upload'
                  ? 'border-primary-600'
                  : trackId
                  ? 'border-gray-400'
                  : 'border-gray-200'
                }
              `}
            >
              1
            </span>
            <span className="ml-2">Upload</span>
          </button>
          <div className="border-t-2 border-gray-200 w-8" />
          <button
            type="button"
            className={`
              flex items-center text-sm font-medium
              ${currentStep === 'metadata' ? 'text-primary-600' : 'text-gray-500'}
              ${!trackId ? 'opacity-50 cursor-not-allowed' : ''}
            `}
            onClick={() => trackId && setCurrentStep('metadata')}
            disabled={!trackId}
          >
            <span
              className={`
                flex-shrink-0 w-8 h-8 flex items-center justify-center rounded-full border-2
                ${currentStep === 'metadata'
                  ? 'border-primary-600'
                  : 'border-gray-200'
                }
              `}
            >
              2
            </span>
            <span className="ml-2">Metadata</span>
          </button>
        </nav>
        <div className="flex-1 border-t-2 border-gray-200" />
      </div>

      <Card>
        {currentStep === 'upload' ? (
          <TrackUpload
            onUpload={handleUpload}
            isUploading={createTrackMutation.isPending}
          />
        ) : (
          <TrackMetadataForm
            onSubmit={handleMetadataSubmit}
            isSubmitting={updateTrackMutation.isPending}
          />
        )}
      </Card>
    </div>
  );
};

export default Upload; 