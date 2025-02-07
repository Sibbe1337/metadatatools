# Metadata Tool Documentation

## Documentation Structure

```plaintext
docs/
├── architecture/           # System architecture documentation
│   ├── overview.md        # High-level architecture overview
│   ├── components.md      # Detailed component documentation
│   └── decisions/         # Architecture decision records
├── development/           # Development guidelines and setup
│   ├── setup.md          # Development environment setup
│   ├── guidelines.md     # Coding standards and guidelines
│   └── workflow.md       # Development workflow
├── api/                  # API documentation
│   ├── endpoints.md      # API endpoints documentation
│   ├── models.md        # Data models
│   └── examples/        # API usage examples
├── deployment/          # Deployment documentation
│   ├── setup.md        # Deployment setup guide
│   ├── monitoring.md   # Monitoring and alerting
│   └── scaling.md      # Scaling guidelines
└── integrations/       # Third-party integrations
    ├── qwen2-audio/   # Qwen2-Audio integration
    ├── openai/        # OpenAI integration
    └── aws/           # AWS services integration

## Quick Links

- [Architecture Overview](architecture/overview.md)
- [Development Setup](development/setup.md)
- [API Documentation](api/endpoints.md)
- [Deployment Guide](deployment/setup.md)
- [Integration Guides](integrations/README.md)

## Contributing to Documentation

Please follow these guidelines when contributing to the documentation:

1. Use Markdown for all documentation files
2. Follow the established folder structure
3. Include code examples where applicable
4. Keep documentation up to date with code changes
5. Add diagrams using Mermaid or PlantUML
6. Reference related documents when appropriate

## Documentation Standards

- Use clear, concise language
- Include examples and use cases
- Keep technical accuracy
- Update related documents
- Use proper formatting and structure
- Include version information where relevant

## Building Documentation

This documentation can be built using [mdBook](https://rust-lang.github.io/mdBook/). To build locally:

```bash
# Install mdBook
cargo install mdbook

# Build documentation
mdbook build

# Serve documentation locally
mdbook serve --open
```

## Documentation Versions

The documentation is versioned alongside the codebase. Each release tag has its corresponding documentation version.

## Getting Help

If you need help with the documentation:

1. Check the existing documentation
2. Look for related architecture decision records
3. Contact the development team
4. Create an issue in the repository

## License

This documentation is licensed under the same terms as the main project. See [LICENSE](../LICENSE) for details. 