import React from 'react';
import { Card } from '../atoms/Card';
import { Button } from '../atoms/Button';
import { TrackMetadata } from './TrackMetadataForm';

interface TrackCardProps {
  track: TrackMetadata & {
    id: string;
    status: 'processing' | 'ready' | 'error';
    aiMetadata?: {
      confidence: number;
      needsReview: boolean;
      reviewReason?: string;
    };
  };
  onEdit?: () => void;
  onDelete?: () => void;
  onExport?: () => void;
}

export const TrackCard: React.FC<TrackCardProps> = ({
  track,
  onEdit,
  onDelete,
  onExport,
}) => {
  const statusColors = {
    processing: 'bg-yellow-100 text-yellow-800',
    ready: 'bg-green-100 text-green-800',
    error: 'bg-red-100 text-red-800',
  };

  return (
    <Card variant="hover" className="relative">
      {/* Status Badge */}
      <div className="absolute top-4 right-4">
        <span
          className={`
            inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium
            ${statusColors[track.status]}
          `}
        >
          {track.status.charAt(0).toUpperCase() + track.status.slice(1)}
        </span>
      </div>

      {/* Track Info */}
      <div className="space-y-4">
        <div>
          <h3 className="text-lg font-medium text-gray-900">{track.title}</h3>
          <p className="text-sm text-gray-500">{track.artist}</p>
        </div>

        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p className="text-gray-500">Genre</p>
            <p className="font-medium">{track.genre}</p>
          </div>
          {track.bpm && (
            <div>
              <p className="text-gray-500">BPM</p>
              <p className="font-medium">{track.bpm}</p>
            </div>
          )}
          {track.key && (
            <div>
              <p className="text-gray-500">Key</p>
              <p className="font-medium">{track.key}</p>
            </div>
          )}
          {track.mood && (
            <div>
              <p className="text-gray-500">Mood</p>
              <p className="font-medium">{track.mood}</p>
            </div>
          )}
        </div>

        {/* AI Metadata */}
        {track.aiMetadata && (
          <div className="border-t pt-4">
            <div className="flex items-center justify-between">
              <p className="text-sm text-gray-500">
                AI Confidence: {(track.aiMetadata.confidence * 100).toFixed(1)}%
              </p>
              {track.aiMetadata.needsReview && (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                  Needs Review
                </span>
              )}
            </div>
            {track.aiMetadata.needsReview && track.aiMetadata.reviewReason && (
              <p className="mt-1 text-sm text-gray-500">
                {track.aiMetadata.reviewReason}
              </p>
            )}
          </div>
        )}

        {/* Actions */}
        <div className="border-t pt-4 flex justify-end space-x-2">
          {onExport && (
            <Button
              variant="secondary"
              size="sm"
              onClick={onExport}
            >
              Export
            </Button>
          )}
          {onEdit && (
            <Button
              variant="secondary"
              size="sm"
              onClick={onEdit}
            >
              Edit
            </Button>
          )}
          {onDelete && (
            <Button
              variant="danger"
              size="sm"
              onClick={onDelete}
            >
              Delete
            </Button>
          )}
        </div>
      </div>
    </Card>
  );
}; 