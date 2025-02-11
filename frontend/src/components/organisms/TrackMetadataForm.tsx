import React from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { TrackMetadata, trackMetadataSchema } from '../../types/track';
import { Button } from '../atoms/Button';
import { Input } from '../atoms/Input';
import { Select } from '../atoms/Select';

interface TrackMetadataFormProps {
  initialData?: Partial<TrackMetadata>;
  onSubmit: (data: TrackMetadata) => void;
  isLoading?: boolean;
}

const genres = [
  'Pop',
  'Rock',
  'Hip Hop',
  'R&B',
  'Electronic',
  'Jazz',
  'Classical',
  'Country',
  'Folk',
  'Blues',
  'Metal',
  'Reggae',
  'World',
  'Other',
];

const musicalKeys = [
  'C',
  'C#/Db',
  'D',
  'D#/Eb',
  'E',
  'F',
  'F#/Gb',
  'G',
  'G#/Ab',
  'A',
  'A#/Bb',
  'B',
];

const moods = [
  'Happy',
  'Sad',
  'Energetic',
  'Calm',
  'Aggressive',
  'Romantic',
  'Dark',
  'Uplifting',
  'Melancholic',
  'Mysterious',
];

export const TrackMetadataForm: React.FC<TrackMetadataFormProps> = ({
  initialData,
  onSubmit,
  isLoading,
}) => {
  const {
    register,
    handleSubmit,
    formState: { errors },
    watch,
  } = useForm<TrackMetadata>({
    resolver: zodResolver(trackMetadataSchema),
    defaultValues: initialData,
  });

  const tags = watch('tags') || [];

  const handleAddTag = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      event.preventDefault();
      const input = event.currentTarget;
      const value = input.value.trim();

      if (value && !tags.includes(value)) {
        const newTags = [...tags, value];
        // @ts-ignore - react-hook-form types don't handle array fields well
        register('tags').onChange({ target: { value: newTags } });
        input.value = '';
      }
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    const newTags = tags.filter((tag) => tag !== tagToRemove);
    // @ts-ignore - react-hook-form types don't handle array fields well
    register('tags').onChange({ target: { value: newTags } });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
        <div>
          <Input
            label="Title"
            {...register('title')}
            error={errors.title?.message}
            required
          />
        </div>
        <div>
          <Input
            label="Artist"
            {...register('artist')}
            error={errors.artist?.message}
            required
          />
        </div>
        <div>
          <Select
            label="Genre"
            {...register('genre')}
            error={errors.genre?.message}
            required
          >
            <option value="">Select a genre</option>
            {genres.map((genre) => (
              <option key={genre} value={genre}>
                {genre}
              </option>
            ))}
          </Select>
        </div>
        <div>
          <Input
            type="number"
            label="BPM"
            {...register('bpm', { valueAsNumber: true })}
            error={errors.bpm?.message}
          />
        </div>
        <div>
          <Select label="Key" {...register('key')} error={errors.key?.message}>
            <option value="">Select a key</option>
            {musicalKeys.map((key) => (
              <option key={key} value={key}>
                {key}
              </option>
            ))}
          </Select>
        </div>
        <div>
          <Select label="Mood" {...register('mood')} error={errors.mood?.message}>
            <option value="">Select a mood</option>
            {moods.map((mood) => (
              <option key={mood} value={mood}>
                {mood}
              </option>
            ))}
          </Select>
        </div>
        <div className="sm:col-span-2">
          <Input
            label="ISRC"
            {...register('isrc')}
            error={errors.isrc?.message}
            placeholder="e.g., USRC17607839"
          />
        </div>
        <div className="sm:col-span-2">
          <label className="block text-sm font-medium text-gray-700">Tags</label>
          <div className="mt-1">
            <Input
              placeholder="Type a tag and press Enter"
              onKeyDown={handleAddTag}
            />
          </div>
          {tags.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-2">
              {tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-primary-100 text-primary-800"
                >
                  {tag}
                  <button
                    type="button"
                    className="ml-1 inline-flex items-center justify-center w-4 h-4 rounded-full hover:bg-primary-200"
                    onClick={() => handleRemoveTag(tag)}
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="flex justify-end space-x-3">
        <Button type="submit" isLoading={isLoading}>
          Save Changes
        </Button>
      </div>
    </form>
  );
}; 