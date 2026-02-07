#!/bin/bash
# =============================================================================
# Script para crear repositorio en GitHub y subir el código
# Ejecutar desde la raíz del proyecto dockerverse
# =============================================================================

echo "🐳 DockerVerse - GitHub Repository Setup"
echo ""

# Variables
GITHUB_USER="vicolmenares"
REPO_NAME="dockerverse"

echo "📝 Pasos para crear el repositorio:"
echo ""
echo "1. Abre GitHub en tu navegador:"
echo "   https://github.com/new"
echo ""
echo "2. Configura el repositorio:"
echo "   • Repository name: $REPO_NAME"
echo "   • Description: Multi-Host Docker Management Dashboard"
echo "   • Visibility: Public (o Private si prefieres)"
echo "   • NO inicialices con README, .gitignore ni LICENSE"
echo ""
echo "3. Después de crear el repositorio, ejecuta estos comandos:"
echo ""
echo "   git remote add origin https://github.com/$GITHUB_USER/$REPO_NAME.git"
echo "   git push -u origin master"
echo "   git push --tags"
echo ""
echo "4. ¡Listo! Tu repositorio estará en:"
echo "   https://github.com/$GITHUB_USER/$REPO_NAME"
echo ""
