import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { PasswordInput } from '../PasswordInput';

describe('PasswordInput', () => {
  it('renders password input with toggle button', () => {
    render(<PasswordInput name="password" data-testid="password-input" />);
    expect(screen.getByTestId('password-input')).toHaveAttribute('type', 'password');
    expect(screen.getByRole('button', { name: /show password/i })).toBeInTheDocument();
  });

  it('toggles password visibility when button is clicked', () => {
    render(<PasswordInput name="password" data-testid="password-input" />);
    const input = screen.getByTestId('password-input');
    const toggleButton = screen.getByRole('button', { name: /show password/i });

    // Initial state
    expect(input).toHaveAttribute('type', 'password');

    // After first click
    fireEvent.click(toggleButton);
    expect(input).toHaveAttribute('type', 'text');

    // After second click
    fireEvent.click(toggleButton);
    expect(input).toHaveAttribute('type', 'password');
  });

  it('forwards ref correctly', () => {
    const ref = React.createRef<HTMLInputElement>();
    render(<PasswordInput name="password" ref={ref} />);
    expect(ref.current).toBeInstanceOf(HTMLInputElement);
  });

  it('displays error message when error prop is provided', () => {
    const errorMessage = 'Password is required';
    render(<PasswordInput name="password" error={errorMessage} />);
    expect(screen.getByText(errorMessage)).toBeInTheDocument();
  });

  it('applies custom className to input', () => {
    const customClass = 'custom-class';
    render(<PasswordInput name="password" className={customClass} data-testid="password-input" />);
    const input = screen.getByTestId('password-input');
    expect(input).toHaveClass(customClass);
  });

  it('preserves other AuthInput props', () => {
    const label = 'Password';
    const placeholder = 'Enter your password';
    render(
      <PasswordInput
        name="password"
        label={label}
        placeholder={placeholder}
        data-testid="password-input"
      />
    );
    
    expect(screen.getByText(label)).toBeInTheDocument();
    expect(screen.getByTestId('password-input')).toHaveAttribute('placeholder', placeholder);
  });
}); 