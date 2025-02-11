import React from 'react';
import { twMerge } from 'tailwind-merge';

interface AuthHeadingProps {
  children: React.ReactNode;
  className?: string;
  level?: 1 | 2 | 3;
}

/**
 * Heading component for authentication pages with consistent styling
 */
export const AuthHeading: React.FC<AuthHeadingProps> = ({
  children,
  className,
  level = 1,
}) => {
  const baseStyles = 'font-semibold text-gray-900 dark:text-white';
  const sizeStyles = {
    1: 'text-3xl',
    2: 'text-2xl',
    3: 'text-xl',
  };

  const Component = `h${level}` as keyof JSX.IntrinsicElements;

  return (
    <Component className={twMerge(baseStyles, sizeStyles[level], className)}>
      {children}
    </Component>
  );
}; 