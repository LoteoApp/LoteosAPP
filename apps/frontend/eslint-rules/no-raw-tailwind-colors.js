const COLOR_NAMES = [
  'slate', 'gray', 'zinc', 'neutral', 'stone',
  'red', 'orange', 'amber', 'yellow', 'lime', 'green', 'emerald', 'teal',
  'cyan', 'sky', 'blue', 'indigo', 'violet', 'purple', 'fuchsia', 'pink', 'rose',
]

const COLOR_PREFIXES = [
  'bg', 'text', 'border', 'ring-offset', 'ring', 'fill', 'stroke', 'divide',
  'outline', 'decoration', 'from', 'via', 'to', 'accent', 'caret', 'shadow', 'placeholder',
]

const RAW_COLOR_CLASS = new RegExp(
  `^(${COLOR_PREFIXES.join('|')})-(${COLOR_NAMES.join('|')})-\\d{2,3}$`,
)

const CLASS_HELPER_NAMES = new Set(['cn', 'clsx', 'twMerge'])

function findRawColorTokens(text) {
  return text.split(/\s+/).filter((token) => {
    const base = token.split(':').pop()
    return base && RAW_COLOR_CLASS.test(base)
  })
}

function report(context, node, text) {
  for (const token of findRawColorTokens(text)) {
    context.report({
      node,
      messageId: 'rawColor',
      data: { token },
    })
  }
}

function checkClassValue(context, node) {
  if (!node) return

  if (node.type === 'Literal' && typeof node.value === 'string') {
    report(context, node, node.value)
    return
  }

  if (node.type === 'TemplateLiteral') {
    for (const quasi of node.quasis) {
      report(context, quasi, quasi.value.cooked ?? quasi.value.raw)
    }
    return
  }

  if (node.type === 'ConditionalExpression') {
    checkClassValue(context, node.consequent)
    checkClassValue(context, node.alternate)
    return
  }

  if (node.type === 'LogicalExpression') {
    checkClassValue(context, node.left)
    checkClassValue(context, node.right)
    return
  }

  if (node.type === 'ArrayExpression') {
    for (const element of node.elements) checkClassValue(context, element)
    return
  }

  if (node.type === 'ObjectExpression') {
    for (const property of node.properties) {
      if (property.type !== 'Property' || property.computed) continue
      if (property.key.type === 'Literal' && typeof property.key.value === 'string') {
        report(context, property.key, property.key.value)
      } else if (property.key.type === 'Identifier') {
        report(context, property.key, property.key.name)
      }
    }
  }

  // Identifiers, member expressions and other dynamic values are out of
  // scope: this rule only catches statically-readable class names.
}

export default {
  meta: {
    type: 'problem',
    docs: {
      description: 'disallow raw Tailwind palette color utilities in favor of theme tokens',
    },
    schema: [],
    messages: {
      rawColor:
        'Do not use the raw Tailwind class "{{token}}"; use a theme token (e.g. text-destructive, text-muted-foreground, bg-primary) defined in the Tailwind theme instead.',
    },
  },
  create(context) {
    return {
      JSXAttribute(node) {
        if (node.name.name !== 'className' || !node.value) return

        if (node.value.type === 'Literal') {
          checkClassValue(context, node.value)
        } else if (node.value.type === 'JSXExpressionContainer') {
          checkClassValue(context, node.value.expression)
        }
      },
      CallExpression(node) {
        if (node.callee.type !== 'Identifier' || !CLASS_HELPER_NAMES.has(node.callee.name)) return
        for (const arg of node.arguments) checkClassValue(context, arg)
      },
    }
  },
}
