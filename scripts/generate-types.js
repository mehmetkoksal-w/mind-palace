#!/usr/bin/env node
/**
 * Generates TypeScript types from Mind Palace JSON schemas.
 *
 * This script reads the JSON schemas from /schemas and generates
 * corresponding TypeScript interfaces for use in the VS Code extension
 * and other TypeScript consumers.
 *
 * Usage:
 *   node scripts/generate-types.js [output-dir]
 *
 * Default output: ./generated/types.ts
 *
 * Requirements:
 *   npm install json-schema-to-typescript
 */

const fs = require('fs');
const path = require('path');

// Try to import json-schema-to-typescript
let compile;
try {
    compile = require('json-schema-to-typescript').compile;
} catch (e) {
    console.error('Error: json-schema-to-typescript not installed.');
    console.error('Run: npm install json-schema-to-typescript');
    process.exit(1);
}

const SCHEMAS_DIR = path.join(__dirname, '..', 'schemas');
const DEFAULT_OUTPUT = path.join(__dirname, '..', 'generated', 'types.ts');

// Schema files to process
const SCHEMA_FILES = [
    'palace.schema.json',
    'room.schema.json',
    'playbook.schema.json',
    'context-pack.schema.json',
    'change-signal.schema.json',
    'scan.schema.json',
    'project-profile.schema.json',
];

async function generateTypes(outputPath) {
    console.log('Generating TypeScript types from JSON schemas...\n');

    const types = [];

    // Add header
    types.push(`/**
 * Auto-generated TypeScript types from Mind Palace JSON schemas.
 * DO NOT EDIT MANUALLY - regenerate with: npm run generate-types
 *
 * Generated: ${new Date().toISOString()}
 */

`);

    for (const schemaFile of SCHEMA_FILES) {
        const schemaPath = path.join(SCHEMAS_DIR, schemaFile);

        if (!fs.existsSync(schemaPath)) {
            console.warn(`  Warning: ${schemaFile} not found, skipping`);
            continue;
        }

        console.log(`  Processing ${schemaFile}...`);

        try {
            const schemaContent = fs.readFileSync(schemaPath, 'utf-8');
            const schema = JSON.parse(schemaContent);

            // Generate TypeScript
            const ts = await compile(schema, schema.title || schemaFile.replace('.schema.json', ''), {
                bannerComment: '',
                additionalProperties: false,
                strictIndexSignatures: true,
            });

            types.push(`// From: ${schemaFile}\n`);
            types.push(ts);
            types.push('\n');
        } catch (error) {
            console.error(`  Error processing ${schemaFile}: ${error.message}`);
        }
    }

    // Ensure output directory exists
    const outputDir = path.dirname(outputPath);
    if (!fs.existsSync(outputDir)) {
        fs.mkdirSync(outputDir, { recursive: true });
    }

    // Write output
    fs.writeFileSync(outputPath, types.join(''));
    console.log(`\nTypes written to: ${outputPath}`);
}

// Main
const outputPath = process.argv[2] || DEFAULT_OUTPUT;
generateTypes(outputPath).catch(error => {
    console.error('Failed to generate types:', error);
    process.exit(1);
});
