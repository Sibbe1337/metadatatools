import React from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Input } from '../atoms/Input';
import { Button } from '../atoms/Button';

const trackMetadataSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  artist: z.string().min(1, 'Artist is required'),
  genre: z.string().min(1, 'Genre is required'),
  bpm: z.number().min(1, 'BPM must be greater than 0').optional(),
  key: z.string().optional(),
  mood: z.string().optional(),
  isrc: z.string().regex(/^[A-Z]{2}[A-Z0-9]{3}[0-9]{7}$/, 'Invalid ISRC format').optional(),
  tags: z.array(z.string()).optional(),
});

export type TrackMetadata = z.infer<typeof trackMetadataSchema>;

interface TrackMetadataFormProps {
  initialData?: Partial<TrackMetadata>;
  onSubmit: (data: TrackMetadata) => Promise<void>;
  isSubmitting?: boolean;
}

export const TrackMetadataForm: React.FC<TrackMetadataFormProps> = ({
  initialData,
  onSubmit,
  isSubmitting,
}) => {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<TrackMetadata>({
    resolver: zodResolver(trackMetadataSchema),
    defaultValues: initialData,
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
        <Input
          label="Title"
          {...register('title')}
          error={errors.title?.message}
        />
        <Input
          label="Artist"
          {...register('artist')}
          error={errors.artist?.message}
        />
        <Input
          label="Genre"
          {...register('genre')}
          error={errors.genre?.message}
        />
        <Input
          label="BPM"
          type="number"
          {...register('bpm', { valueAsNumber: true })}
          error={errors.bpm?.message}
        />
        <Input
          label="Key"
          {...register('key')}
          error={errors.key?.message}
        />
        <Input
          label="Mood"
          {...register('mood')}
          error={errors.mood?.message}
        />
        <Input
          label="ISRC"
          {...register('isrc')}
          error={errors.isrc?.message}
          helperText="Format: CC-XXX-YY-NNNNN"
        />
        <Input
          label="Tags"
          {...register('tags')}
          error={errors.tags?.message}
          helperText="Comma-separated list of tags"
        />
      </div>
      <div className="flex justify-end space-x-4">
        <Button
          type="button"
          variant="secondary"
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          isLoading={isSubmitting}
        >
          Save Changes
        </Button>
      </div>
    </form>
  );
}; 