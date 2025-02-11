import React from 'react';
import { twMerge } from 'tailwind-merge';

interface FormFieldProps {
  children: React.ReactNode;
  label?: string;
  error?: string;
  helperText?: string;
  required?: boolean;
  className?: string;
  labelClassName?: string;
}

/**
 * Form field wrapper component for consistent layout and error handling
 */
export const FormField: React.FC<FormFieldProps> = ({
  children,
  label,
  error,
  helperText,
  required,
  className,
  labelClassName,
}) => {
  const id = React.Children.only(children as React.ReactElement).props.id ||
            React.Children.only(children as React.ReactElement).props.name;

  return (
    <div className={twMerge('space-y-2', className)} data-testid="form-field">
      {label && (
        <label
          htmlFor={id}
          className={twMerge(
            'block text-sm font-medium text-gray-900 dark:text-white',
            labelClassName
          )}
        >
          {label}
          {required && (
            <span className="ml-1 text-red-500" aria-hidden="true">
              *
            </span>
          )}
        </label>
      )}
      {children}
      {error && (
        <p
          className="text-sm text-red-600 dark:text-red-400"
          id={`${id}-error`}
          role="alert"
        >
          {error}
        </p>
      )}
      {!error && helperText && (
        <p
          className="text-sm text-gray-500 dark:text-gray-400"
          id={`${id}-helper`}
        >
          {helperText}
        </p>
      )}
    </div>
  );
}; 