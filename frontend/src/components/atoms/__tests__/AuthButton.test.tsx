import { render, screen, fireEvent } from '@testing-library/react';
import { AuthButton } from '../AuthButton';

describe('AuthButton', () => {
  it('renders children correctly', () => {
    render(<AuthButton>Click me</AuthButton>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('applies primary variant styles by default', () => {
    render(<AuthButton>Button</AuthButton>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('bg-blue-600');
  });

  it('applies different variant styles', () => {
    render(<AuthButton variant="secondary">Button</AuthButton>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('bg-gray-600');
  });

  it('shows loading spinner when isLoading is true', () => {
    render(<AuthButton isLoading>Button</AuthButton>);
    expect(screen.getByRole('button')).toBeDisabled();
    expect(screen.getByRole('button').querySelector('svg')).toBeInTheDocument();
  });

  it('applies full width style when fullWidth is true', () => {
    render(<AuthButton fullWidth>Button</AuthButton>);
    expect(screen.getByRole('button')).toHaveClass('w-full');
  });

  it('handles click events', () => {
    const handleClick = jest.fn();
    render(<AuthButton onClick={handleClick}>Button</AuthButton>);
    
    fireEvent.click(screen.getByRole('button'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('does not trigger click when disabled', () => {
    const handleClick = jest.fn();
    render(<AuthButton disabled onClick={handleClick}>Button</AuthButton>);
    
    fireEvent.click(screen.getByRole('button'));
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('does not trigger click when loading', () => {
    const handleClick = jest.fn();
    render(<AuthButton isLoading onClick={handleClick}>Button</AuthButton>);
    
    fireEvent.click(screen.getByRole('button'));
    expect(handleClick).not.toHaveBeenCalled();
  });
}); 