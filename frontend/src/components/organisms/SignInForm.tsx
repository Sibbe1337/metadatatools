import React from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { AuthButton } from '../atoms/AuthButton';
import { AuthInput } from '../atoms/AuthInput';
import { PasswordInput } from '../molecules/PasswordInput';
import { FormField } from '../molecules/FormField';
import { AuthHeading } from '../atoms/AuthHeading';
import { AuthLink } from '../atoms/AuthLink';
import { AuthError } from '../atoms/AuthError';
import { useAuth } from '../../context/AuthContext';
import { AuthErrorType } from '../../types/auth';

const signInSchema = z.object({
  email: z.string().email('Please enter a valid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
});

type SignInFormData = z.infer<typeof signInSchema>;

export const SignInForm: React.FC = () => {
  const { login, error, isLoading } = useAuth();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<SignInFormData>({
    resolver: zodResolver(signInSchema),
  });

  const onSubmit = async (data: SignInFormData) => {
    await login(data);
  };

  return (
    <div className="w-full max-w-md space-y-8">
      <div className="text-center">
        <AuthHeading>Sign in to your account</AuthHeading>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          Don't have an account?{' '}
          <AuthLink to="/auth/register">Create one here</AuthLink>
        </p>
      </div>

      {error && (
        <AuthError
          type={error.type as AuthErrorType}
          message={error.message}
        />
      )}

      <form className="space-y-6" onSubmit={handleSubmit(onSubmit)} role="form">
        <FormField
          label="Email address"
          error={errors.email?.message}
          required
        >
          <AuthInput
            {...register('email')}
            type="email"
            autoComplete="email"
            placeholder="Enter your email"
            data-testid="email-input"
          />
        </FormField>

        <FormField
          label="Password"
          error={errors.password?.message}
          required
        >
          <PasswordInput
            {...register('password')}
            autoComplete="current-password"
            placeholder="Enter your password"
            data-testid="password-input"
          />
        </FormField>

        <div className="flex items-center justify-between">
          <div className="flex items-center">
            <input
              id="remember-me"
              name="remember-me"
              type="checkbox"
              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              data-testid="remember-me-checkbox"
            />
            <label
              htmlFor="remember-me"
              className="ml-2 block text-sm text-gray-900 dark:text-gray-300"
            >
              Remember me
            </label>
          </div>

          <AuthLink
            to="/auth/forgot-password"
            variant="secondary"
            className="text-sm"
          >
            Forgot your password?
          </AuthLink>
        </div>

        <AuthButton
          type="submit"
          fullWidth
          isLoading={isLoading}
        >
          Sign in
        </AuthButton>
      </form>
    </div>
  );
}; 