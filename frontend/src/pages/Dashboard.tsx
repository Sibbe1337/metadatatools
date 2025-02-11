import React from 'react';
import { Link } from 'react-router-dom';
import { Card } from '../components/atoms/Card';
import { Button } from '../components/atoms/Button';
import { useTrackApi } from '../hooks/useTrackApi';
import {
  ChartBarIcon,
  CloudArrowUpIcon,
  DocumentCheckIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline';

const Dashboard = () => {
  const { useTracks } = useTrackApi();
  const { data: tracks, isLoading } = useTracks({ limit: 5 });

  const stats = [
    {
      name: 'Total Tracks',
      value: tracks?.length || 0,
      icon: ChartBarIcon,
      color: 'bg-blue-500',
    },
    {
      name: 'Processing',
      value: tracks?.filter(t => t.status === 'processing').length || 0,
      icon: CloudArrowUpIcon,
      color: 'bg-yellow-500',
    },
    {
      name: 'Needs Review',
      value: tracks?.filter(t => t.aiMetadata?.needsReview).length || 0,
      icon: ExclamationTriangleIcon,
      color: 'bg-red-500',
    },
    {
      name: 'Validated',
      value: tracks?.filter(t => t.status === 'ready' && !t.aiMetadata?.needsReview).length || 0,
      icon: DocumentCheckIcon,
      color: 'bg-green-500',
    },
  ];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Welcome to Audio Metadata Magic</h1>
        <p className="mt-2 text-lg text-gray-600">
          Enhance your audio tracks with AI-powered metadata enrichment
        </p>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Link to="/upload">
          <Card variant="hover" className="h-full">
            <div className="flex items-center space-x-4 p-6">
              <div className="bg-primary-100 rounded-lg p-3">
                <CloudArrowUpIcon className="h-6 w-6 text-primary-600" />
              </div>
              <div>
                <h3 className="text-lg font-medium text-gray-900">Upload New Track</h3>
                <p className="text-sm text-gray-500">Upload and analyze your audio files</p>
              </div>
            </div>
          </Card>
        </Link>
        <Link to="/tracks">
          <Card variant="hover" className="h-full">
            <div className="flex items-center space-x-4 p-6">
              <div className="bg-primary-100 rounded-lg p-3">
                <ChartBarIcon className="h-6 w-6 text-primary-600" />
              </div>
              <div>
                <h3 className="text-lg font-medium text-gray-900">View All Tracks</h3>
                <p className="text-sm text-gray-500">Manage and export your track metadata</p>
              </div>
            </div>
          </Card>
        </Link>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat) => (
          <Card key={stat.name} className="relative overflow-hidden">
            <div className="absolute top-0 right-0 -mt-4 -mr-4 w-24 h-24 opacity-10">
              <stat.icon className={`w-full h-full ${stat.color}`} />
            </div>
            <div className="p-6">
              <dt className="text-sm font-medium text-gray-500 truncate">{stat.name}</dt>
              <dd className="mt-2 text-3xl font-semibold text-gray-900">{stat.value}</dd>
            </div>
          </Card>
        ))}
      </div>

      {/* Recent Tracks */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-medium text-gray-900">Recent Tracks</h2>
          <Link to="/tracks">
            <Button variant="secondary" size="sm">View All</Button>
          </Link>
        </div>
        <Card>
          {isLoading ? (
            <div className="animate-pulse space-y-4 p-6">
              {[...Array(3)].map((_, i) => (
                <div key={i} className="h-16 bg-gray-200 rounded" />
              ))}
            </div>
          ) : tracks?.length ? (
            <div className="divide-y divide-gray-200">
              {tracks.map((track) => (
                <div key={track.id} className="p-4 hover:bg-gray-50">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="text-sm font-medium text-gray-900">
                        {track.metadata.title}
                      </h3>
                      <p className="text-sm text-gray-500">{track.metadata.artist}</p>
                    </div>
                    <div className="flex items-center space-x-2">
                      {track.aiMetadata?.needsReview && (
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                          Needs Review
                        </span>
                      )}
                      <span
                        className={`
                          inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium
                          ${track.status === 'ready'
                            ? 'bg-green-100 text-green-800'
                            : track.status === 'processing'
                            ? 'bg-yellow-100 text-yellow-800'
                            : 'bg-red-100 text-red-800'
                          }
                        `}
                      >
                        {track.status.charAt(0).toUpperCase() + track.status.slice(1)}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <CloudArrowUpIcon className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No tracks yet</h3>
              <p className="mt-1 text-sm text-gray-500">Get started by uploading your first track.</p>
              <div className="mt-6">
                <Link to="/upload">
                  <Button>Upload Track</Button>
                </Link>
              </div>
            </div>
          )}
        </Card>
      </div>
    </div>
  );
};

export default Dashboard;