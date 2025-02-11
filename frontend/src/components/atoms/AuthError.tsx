import React from 'react';
import { twMerge } from 'tailwind-merge';
import { AuthErrorType } from '../../types/auth';

interface AuthErrorProps {
  type: AuthErrorType;
  message: string;
  className?: string;
}

/**
 * Error message component for authentication pages
 */
export const AuthError: React.FC<AuthErrorProps> = ({
  type,
  message,
  className,
}) => {
  const baseStyles = 'flex items-center p-4 mb-4 text-sm rounded-lg';
  const errorStyles = 'text-red-800 bg-red-50 dark:bg-red-900/10 dark:text-red-400';

  const getIcon = () => {
    return (
      <svg
        className="w-4 h-4 mr-2 flex-shrink-0"
        aria-hidden="true"
        xmlns="http://www.w3.org/2000/svg"
        fill="currentColor"
        viewBox="0 0 20 20"
      >
        <path d="M10 .5a9.5 9.5 0 1 0 9.5 9.5A9.51 9.51 0 0 0 10 .5ZM10 15a1 1 0 1 1 0-2 1 1 0 0 1 0 2Zm1-4a1 1 0 0 1-2 0V6a1 1 0 0 1 2 0v5Z"/>
      </svg>
    );
  };

  return (
    <div
      role="alert"
      className={twMerge(baseStyles, errorStyles, className)}
      data-error-type={type}
    >
      {getIcon()}
      <span className="sr-only">Error:</span>
      <span>{message}</span>
    </div>
  );
}; 