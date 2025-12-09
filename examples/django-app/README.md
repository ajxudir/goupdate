# Django App Example

This example demonstrates goupdate configuration for a Python/Django project with system tests.

## Key Concepts

1. **Package groups** - Group web, async, database, and testing dependencies
2. **Ignore patterns** - Skip formatters that shouldn't auto-update
3. **Version exclusions** - Filter out dev/post releases
4. **System tests** - Django checks, pytest, migrations validation

## Configuration Highlights

```yaml
rules:
  requirements:
    groups:
      web: [Django, djangorestframework, gunicorn]
      async: [celery, redis]
      database: [psycopg2-binary]
      testing: [pytest, pytest-django]

    ignore: [black, isort]  # Don't update formatters
    exclude_versions:
      - "(?i)dev"
      - "(?i)post"

system_tests:
  run_preflight: true
  tests:
    - name: django-check
      commands: python manage.py check
    - name: unit-tests
      commands: pytest -v --tb=short
    - name: migrations-check
      commands: python manage.py migrate --check
```

## Try It

```bash
cd examples/django-app
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt
goupdate scan              # Discover requirements.txt
goupdate list              # Show declared versions
goupdate outdated          # Check for updates
goupdate update --dry-run  # Preview changes
```

## Why Ignore Formatters?

Code formatters like `black` and `isort` are pinned by teams to ensure consistent formatting. Auto-updating them could change formatting rules mid-project.

## Why System Tests?

Django's `manage.py check` catches configuration issues. Running pytest validates that updates don't break tests. Migration checks ensure schema compatibility.
