import React from 'react';
import { Link } from 'react-router-dom';
import { twMerge } from 'tailwind-merge';

interface AuthLinkProps {
  children: React.ReactNode;
  to: string;
  className?: string;
  variant?: 'primary' | 'secondary';
}

/**
 * Link component for authentication pages with consistent styling
 */
export const AuthLink: React.FC<AuthLinkProps> = ({
  children,
  to,
  className,
  variant = 'primary',
}) => {
  const baseStyles = 'text-sm font-medium transition-colors duration-200';
  const variantStyles = {
    primary: 'text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300',
    secondary: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white',
  };

  return (
    <Link
      to={to}
      className={twMerge(baseStyles, variantStyles[variant], className)}
    >
      {children}
    </Link>
  );
}; 