import React, { useCallback } from 'react';
import { useDropzone } from 'react-dropzone';
import { CloudArrowUpIcon } from '@heroicons/react/24/outline';

interface TrackUploadProps {
  onUpload: (file: File) => void;
  isLoading?: boolean;
  accept?: string[];
  maxSize?: number;
}

export const TrackUpload: React.FC<TrackUploadProps> = ({
  onUpload,
  isLoading,
  accept = ['audio/mpeg', 'audio/wav', 'audio/aiff', 'audio/flac'],
  maxSize = 100 * 1024 * 1024, // 100MB
}) => {
  const onDrop = useCallback(
    (acceptedFiles: File[]) => {
      if (acceptedFiles.length > 0) {
        onUpload(acceptedFiles[0]);
      }
    },
    [onUpload]
  );

  const { getRootProps, getInputProps, isDragActive, fileRejections } = useDropzone({
    onDrop,
    accept: accept.reduce((acc, curr) => ({ ...acc, [curr]: [] }), {}),
    maxSize,
    multiple: false,
    disabled: isLoading,
  });

  const fileRejectionError = fileRejections[0]?.errors[0]?.message;

  return (
    <div className="space-y-4">
      <div
        {...getRootProps()}
        className={`
          relative block w-full rounded-lg border-2 border-dashed p-12 text-center
          focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2
          ${isDragActive ? 'border-primary-400 bg-primary-50' : 'border-gray-300'}
          ${isLoading ? 'cursor-not-allowed opacity-50' : 'cursor-pointer hover:border-gray-400'}
        `}
      >
        <input {...getInputProps()} />
        <CloudArrowUpIcon
          className={`mx-auto h-12 w-12 ${
            isDragActive ? 'text-primary-400' : 'text-gray-400'
          }`}
        />
        <span className="mt-2 block text-sm font-semibold text-gray-900">
          {isDragActive ? 'Drop the file here' : 'Upload a track'}
        </span>
        <span className="mt-2 block text-sm text-gray-500">
          {isLoading
            ? 'Uploading...'
            : `Drag and drop or click to select a file (max ${(maxSize / 1024 / 1024).toFixed(
                0
              )}MB)`}
        </span>
        <span className="mt-1 block text-xs text-gray-500">
          Supported formats: {accept.join(', ')}
        </span>
      </div>

      {fileRejectionError && (
        <div className="rounded-md bg-red-50 p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg
                className="h-5 w-5 text-red-400"
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                aria-hidden="true"
              >
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z"
                  clipRule="evenodd"
                />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Upload error</h3>
              <div className="mt-2 text-sm text-red-700">
                <p>{fileRejectionError}</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}; 