import React, { useCallback, useState } from 'react';
import { useDropzone } from 'react-dropzone';
import {
  Box,
  Paper,
  Typography,
  LinearProgress,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  IconButton,
  Button,
  Grid,
} from '@mui/material';
import { CloudUpload, CheckCircle, Error, Close } from '@mui/icons-material';

interface UploadFile extends File {
  id: string;
  progress: number;
  status: 'pending' | 'uploading' | 'success' | 'error';
  error?: string;
}

const BatchUpload: React.FC = () => {
  const [files, setFiles] = useState<UploadFile[]>([]);

  const onDrop = useCallback((acceptedFiles: File[]) => {
    const newFiles = acceptedFiles.map(file => ({
      ...file,
      id: Math.random().toString(36).substring(7),
      progress: 0,
      status: 'pending' as const,
    }));
    setFiles(prev => [...prev, ...newFiles]);
  }, []);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'audio/*': ['.mp3', '.wav', '.aiff', '.flac']
    },
    multiple: true
  });

  const removeFile = (id: string) => {
    setFiles(prev => prev.filter(file => file.id !== id));
  };

  const uploadFiles = async () => {
    // TODO: Implement batch upload logic
    console.log('Uploading files:', files);
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Batch Upload
      </Typography>

      <Grid container spacing={3}>
        <Grid item xs={12}>
          <Paper
            elevation={2}
            sx={{
              p: 3,
              backgroundColor: theme => 
                isDragActive ? theme.palette.action.hover : theme.palette.background.paper,
              transition: 'background-color 0.2s ease',
            }}
            {...getRootProps()}
          >
            <input {...getInputProps()} />
            <Box
              sx={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                minHeight: '200px',
                border: 2,
                borderRadius: 1,
                borderColor: 'divider',
                borderStyle: 'dashed',
                p: 3,
              }}
            >
              <CloudUpload sx={{ fontSize: 48, mb: 2, color: 'primary.main' }} />
              <Typography variant="h6" align="center" gutterBottom>
                {isDragActive
                  ? 'Drop the files here...'
                  : 'Drag and drop audio files here, or click to select files'}
              </Typography>
              <Typography variant="body2" color="textSecondary" align="center">
                Supported formats: MP3, WAV, AIFF, FLAC
              </Typography>
            </Box>
          </Paper>
        </Grid>

        {files.length > 0 && (
          <Grid item xs={12}>
            <Paper elevation={2}>
              <Box sx={{ p: 3 }}>
                <Typography variant="h6" gutterBottom>
                  Upload Queue ({files.length} files)
                </Typography>
                <List>
                  {files.map((file) => (
                    <ListItem
                      key={file.id}
                      secondaryAction={
                        <IconButton edge="end" onClick={() => removeFile(file.id)}>
                          <Close />
                        </IconButton>
                      }
                    >
                      <ListItemIcon>
                        {file.status === 'success' ? (
                          <CheckCircle color="success" />
                        ) : file.status === 'error' ? (
                          <Error color="error" />
                        ) : (
                          <CloudUpload color="primary" />
                        )}
                      </ListItemIcon>
                      <ListItemText
                        primary={file.name}
                        secondary={
                          file.error || `${(file.size / 1024 / 1024).toFixed(2)} MB`
                        }
                      />
                      {file.status === 'uploading' && (
                        <Box sx={{ width: '100px', ml: 2 }}>
                          <LinearProgress
                            variant="determinate"
                            value={file.progress}
                          />
                        </Box>
                      )}
                    </ListItem>
                  ))}
                </List>
                <Box sx={{ mt: 2, display: 'flex', justifyContent: 'flex-end' }}>
                  <Button
                    variant="contained"
                    color="primary"
                    onClick={uploadFiles}
                    disabled={files.length === 0}
                  >
                    Upload All
                  </Button>
                </Box>
              </Box>
            </Paper>
          </Grid>
        )}
      </Grid>
    </Box>
  );
};

export default BatchUpload; 