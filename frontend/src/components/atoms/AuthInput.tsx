import React, { forwardRef } from 'react';
import { twMerge } from 'tailwind-merge';

interface AuthInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  error?: string;
  label?: string;
  helperText?: string;
}

/**
 * Input component for authentication forms with consistent styling
 */
export const AuthInput = forwardRef<HTMLInputElement, AuthInputProps>(
  ({ className, error, label, helperText, id, ...props }, ref) => {
    const inputId = id || props.name;
    
    const baseStyles = 'w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 transition-colors duration-200';
    const validStyles = 'border-gray-300 focus:border-blue-500 focus:ring-blue-500/20 dark:border-gray-600 dark:bg-gray-800 dark:text-white';
    const errorStyles = 'border-red-500 focus:border-red-500 focus:ring-red-500/20 dark:border-red-500';
    const disabledStyles = 'bg-gray-100 cursor-not-allowed dark:bg-gray-700';

    const inputStyles = twMerge(
      baseStyles,
      props.disabled ? disabledStyles : error ? errorStyles : validStyles,
      className
    );

    return (
      <div className="w-full">
        {label && (
          <label
            htmlFor={inputId}
            className="block mb-2 text-sm font-medium text-gray-900 dark:text-white"
          >
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          className={inputStyles}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={error ? `${inputId}-error` : helperText ? `${inputId}-helper` : undefined}
          {...props}
        />
        {error && (
          <p
            id={`${inputId}-error`}
            className="mt-1 text-sm text-red-600 dark:text-red-400"
          >
            {error}
          </p>
        )}
        {!error && helperText && (
          <p
            id={`${inputId}-helper`}
            className="mt-1 text-sm text-gray-500 dark:text-gray-400"
          >
            {helperText}
          </p>
        )}
      </div>
    );
  }
); 