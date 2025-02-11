import React from 'react';
import { render, screen } from '@testing-library/react';
import { AuthInput } from '../AuthInput';

describe('AuthInput', () => {
  it('renders input with label correctly', () => {
    render(<AuthInput label="Email" name="email" />);
    expect(screen.getByLabelText('Email')).toBeInTheDocument();
  });

  it('displays error message when error prop is provided', () => {
    const errorMessage = 'This field is required';
    render(<AuthInput name="email" error={errorMessage} />);
    expect(screen.getByText(errorMessage)).toBeInTheDocument();
    expect(screen.getByRole('textbox')).toHaveAttribute('aria-invalid', 'true');
  });

  it('displays helper text when provided and no error', () => {
    const helperText = 'Enter your email address';
    render(<AuthInput name="email" helperText={helperText} />);
    expect(screen.getByText(helperText)).toBeInTheDocument();
  });

  it('prioritizes error message over helper text', () => {
    const helperText = 'Enter your email address';
    const errorMessage = 'This field is required';
    render(<AuthInput name="email" helperText={helperText} error={errorMessage} />);
    expect(screen.getByText(errorMessage)).toBeInTheDocument();
    expect(screen.queryByText(helperText)).not.toBeInTheDocument();
  });

  it('applies disabled styles when disabled', () => {
    render(<AuthInput name="email" disabled />);
    expect(screen.getByRole('textbox')).toHaveClass('bg-gray-100');
    expect(screen.getByRole('textbox')).toBeDisabled();
  });

  it('forwards ref correctly', () => {
    const ref = React.createRef<HTMLInputElement>();
    render(<AuthInput name="email" ref={ref} />);
    expect(ref.current).toBeInstanceOf(HTMLInputElement);
  });

  it('applies custom className correctly', () => {
    const customClass = 'custom-class';
    render(<AuthInput name="email" className={customClass} />);
    expect(screen.getByRole('textbox')).toHaveClass(customClass);
  });
}); 