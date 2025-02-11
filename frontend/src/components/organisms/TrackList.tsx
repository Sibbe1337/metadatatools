import React from 'react';
import { Link } from 'react-router-dom';
import {
  PencilIcon,
  TrashIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
  ClockIcon,
} from '@heroicons/react/24/outline';
import { Track } from '../../types/track';
import { Button } from '../atoms/Button';

interface TrackListProps {
  tracks: Track[];
  isLoading?: boolean;
  onEdit?: (track: Track) => void;
  onDelete?: (track: Track) => void;
}

const statusIcons = {
  processing: ClockIcon,
  ready: CheckCircleIcon,
  error: ExclamationTriangleIcon,
};

const statusColors = {
  processing: 'text-yellow-500',
  ready: 'text-green-500',
  error: 'text-red-500',
};

export const TrackList: React.FC<TrackListProps> = ({
  tracks,
  isLoading,
  onEdit,
  onDelete,
}) => {
  if (isLoading) {
    return (
      <div className="space-y-4">
        {[...Array(3)].map((_, i) => (
          <div
            key={i}
            className="animate-pulse bg-white rounded-lg shadow-sm border border-gray-200 p-4"
          >
            <div className="h-6 bg-gray-200 rounded w-1/4 mb-2" />
            <div className="h-4 bg-gray-200 rounded w-1/3" />
          </div>
        ))}
      </div>
    );
  }

  if (!tracks.length) {
    return (
      <div className="text-center py-12 bg-white rounded-lg shadow-sm border border-gray-200">
        <ExclamationTriangleIcon className="mx-auto h-12 w-12 text-gray-400" />
        <h3 className="mt-2 text-sm font-medium text-gray-900">No tracks found</h3>
        <p className="mt-1 text-sm text-gray-500">
          Try adjusting your search or filters to find what you're looking for.
        </p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 divide-y divide-gray-200">
      {tracks.map((track) => {
        const StatusIcon = statusIcons[track.status];

        return (
          <div key={track.id} className="p-4 sm:p-6">
            <div className="flex items-center justify-between">
              <div className="flex-1 min-w-0">
                <div className="flex items-center space-x-3">
                  <h2 className="text-lg font-medium text-gray-900 truncate">
                    {track.metadata.title}
                  </h2>
                  <div className="flex items-center space-x-2">
                    <StatusIcon
                      className={`h-5 w-5 ${statusColors[track.status]}`}
                      aria-hidden="true"
                    />
                    {track.aiMetadata?.needsReview && (
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                        Needs Review
                      </span>
                    )}
                  </div>
                </div>
                <div className="mt-1 flex items-center space-x-2 text-sm text-gray-500">
                  <span>{track.metadata.artist}</span>
                  <span>•</span>
                  <span>{track.metadata.genre}</span>
                  {track.metadata.bpm && (
                    <>
                      <span>•</span>
                      <span>{track.metadata.bpm} BPM</span>
                    </>
                  )}
                  {track.metadata.key && (
                    <>
                      <span>•</span>
                      <span>{track.metadata.key}</span>
                    </>
                  )}
                </div>
                {track.metadata.tags.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-2">
                    {track.metadata.tags.map((tag) => (
                      <span
                        key={tag}
                        className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                )}
              </div>
              <div className="ml-4 flex items-center space-x-2">
                <Link to={`/tracks/${track.id}`}>
                  <Button
                    variant="secondary"
                    size="sm"
                    leftIcon={<PencilIcon className="h-4 w-4" />}
                    onClick={() => onEdit?.(track)}
                  >
                    Edit
                  </Button>
                </Link>
                <Button
                  variant="danger"
                  size="sm"
                  leftIcon={<TrashIcon className="h-4 w-4" />}
                  onClick={() => onDelete?.(track)}
                >
                  Delete
                </Button>
              </div>
            </div>
            {track.aiMetadata?.needsReview && (
              <div className="mt-4 bg-yellow-50 p-4 rounded-md">
                <div className="flex">
                  <div className="flex-shrink-0">
                    <ExclamationTriangleIcon
                      className="h-5 w-5 text-yellow-400"
                      aria-hidden="true"
                    />
                  </div>
                  <div className="ml-3">
                    <h3 className="text-sm font-medium text-yellow-800">Review Required</h3>
                    <div className="mt-2 text-sm text-yellow-700">
                      <p>{track.aiMetadata.reviewReason}</p>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}; 