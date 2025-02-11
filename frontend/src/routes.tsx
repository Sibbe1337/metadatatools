import React from 'react';
import { RouteObject } from 'react-router-dom';
import { Layout } from './components/templates/Layout';

// Lazy load pages
const Dashboard = React.lazy(() => import('./pages/Dashboard'));
const Tracks = React.lazy(() => import('./pages/Tracks'));
const TrackDetails = React.lazy(() => import('./pages/TrackDetails'));
const Upload = React.lazy(() => import('./pages/Upload'));
const Settings = React.lazy(() => import('./pages/Settings'));
const BatchUpload = React.lazy(() => import('./pages/BatchUpload'));
const AISettings = React.lazy(() => import('./pages/AISettings'));

export const routes: RouteObject[] = [
  {
    path: '/',
    element: <Layout />,
    children: [
      {
        index: true,
        element: <Dashboard />,
      },
      {
        path: 'tracks',
        children: [
          {
            index: true,
            element: <Tracks />,
          },
          {
            path: ':id',
            element: <TrackDetails />,
          },
        ],
      },
      {
        path: 'upload',
        children: [
          {
            index: true,
            element: <Upload />,
          },
          {
            path: 'batch',
            element: <BatchUpload />,
          },
        ],
      },
      {
        path: 'settings',
        children: [
          {
            index: true,
            element: <Settings />,
          },
          {
            path: 'ai',
            element: <AISettings />,
          },
        ],
      },
    ],
  },
]; 