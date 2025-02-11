import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Track } from '../types/track';
import { Box, Typography, CircularProgress, Paper, Grid } from '@mui/material';

const TrackDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [track, setTrack] = useState<Track | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchTrackDetails = async () => {
      try {
        setLoading(true);
        // TODO: Implement API call to fetch track details
        // const response = await api.get(`/tracks/${id}`);
        // setTrack(response.data);
        setLoading(false);
      } catch (err) {
        setError('Failed to load track details');
        setLoading(false);
      }
    };

    if (id) {
      fetchTrackDetails();
    }
  }, [id]);

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="60vh">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="60vh">
        <Typography color="error">{error}</Typography>
      </Box>
    );
  }

  if (!track) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="60vh">
        <Typography>Track not found</Typography>
      </Box>
    );
  }

  return (
    <Box p={3}>
      <Paper elevation={2}>
        <Box p={3}>
          <Grid container spacing={3}>
            <Grid item xs={12}>
              <Typography variant="h4" gutterBottom>
                {track.metadata.title}
              </Typography>
            </Grid>
            <Grid item xs={12} md={6}>
              <Typography variant="subtitle1">
                <strong>Artist:</strong> {track.metadata.artist}
              </Typography>
              <Typography variant="subtitle1">
                <strong>Genre:</strong> {track.metadata.genre}
              </Typography>
              <Typography variant="subtitle1">
                <strong>Tags:</strong> {track.metadata.tags.join(', ')}
              </Typography>
            </Grid>
            <Grid item xs={12} md={6}>
              <Typography variant="subtitle1">
                <strong>BPM:</strong> {track.metadata.bpm || 'N/A'}
              </Typography>
              <Typography variant="subtitle1">
                <strong>Key:</strong> {track.metadata.key || 'N/A'}
              </Typography>
              <Typography variant="subtitle1">
                <strong>ISRC:</strong> {track.metadata.isrc || 'N/A'}
              </Typography>
              <Typography variant="subtitle1">
                <strong>Status:</strong> {track.status}
              </Typography>
            </Grid>
            {track.aiMetadata && (
              <Grid item xs={12}>
                <Typography variant="h6" gutterBottom>
                  AI Analysis
                </Typography>
                <Typography variant="subtitle1">
                  <strong>Confidence:</strong> {track.aiMetadata.confidence || 'N/A'}
                </Typography>
                <Typography variant="subtitle1">
                  <strong>Model:</strong> {track.aiMetadata.model}
                </Typography>
                <Typography variant="subtitle1">
                  <strong>Processed At:</strong> {new Date(track.aiMetadata.processedAt).toLocaleString()}
                </Typography>
                {track.aiMetadata.needsReview && (
                  <Typography variant="subtitle1" color="warning.main">
                    <strong>Needs Review:</strong> {track.aiMetadata.reviewReason || 'Unknown reason'}
                  </Typography>
                )}
              </Grid>
            )}
          </Grid>
        </Box>
      </Paper>
    </Box>
  );
};

export default TrackDetails; 