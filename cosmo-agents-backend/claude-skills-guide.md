# Claude Skills Usage Guide for Cosmo Backend

## 🚀 Quick Start Commands

### **Feature Development Workflow**

#### **1. Architecture Phase**
```bash
# Clone and adapt this command:
"Design <feature_name> using cosmo-architecture-patterns"

# Examples:
"Design WhatsApp integration using cosmo-architecture-patterns"
"Design email campaign feature using cosmo-architecture-patterns"
"Design AI pipeline for content generation using cosmo-architecture-patterns"
```

#### **2. Implementation Phase**
```bash
# Clone and adapt this command:
"Implement <feature_name> with test-driven-development and rock-test"

# Examples:
"Implement WhatsApp integration with test-driven-development and rock-test"
"Implement email template system with comprehensive testing"
"Implement user authentication with TDD methodology"
```

#### **3. Code Review Phase**
```bash
# Clone and adapt this command:
"Review <feature/module> using rock-review skill"

# Examples:
"Review authentication code using rock-review skill"
"Security review for payment system with rock-review"
"Performance evaluation for database queries"
```

#### **4. Testing Phase**
```bash
# Clone and adapt this command:
"Write comprehensive tests for <module> with rock-test"

# Examples:
"Write comprehensive tests for email service with rock-test"
"Create integration tests for WhatsApp endpoints"
"Generate performance tests for AI service"
```

#### **5. Deployment Phase**
```bash
# Clone and adapt this command:
"Deploy <feature> to <environment> with rock-deploy"

# Examples:
"Deploy email feature to staging with rock-deploy"
"Deploy authentication update to production with zero-downtime"
"Database migration for new schema with rock-deploy"
```

## 📋 Specific Task Templates

### **Bug Fix Workflow**
```bash
# Fix bug with testing:
"Fix <bug_description> with comprehensive unit tests and integration tests"
"Debug <issue> with proper error handling and test coverage"
"Resolve <problem> with root cause analysis and preventive tests"
```

### **API Development**
```bash
# New API endpoint:
"Create <endpoint> API with comprehensive testing, validation, and documentation"
"Implement <resource> CRUD operations with proper error handling"
"Design RESTful API for <domain> according to Cosmo standards"
```

### **Database Operations**
```bash
# Database changes:
"Design database schema for <feature> with proper indexing and migrations"
"Optimize <query> performance with proper indexing strategies"
"Create database migration for <schema_change> with rollback strategy"
```

### **Security Tasks**
```bash
# Security focused:
"Security audit for <module> with comprehensive vulnerability assessment"
"Implement authentication enhancement for <feature>"
"Add input validation and sanitization for <endpoint>"
```

### **Performance Tasks**
```bash
# Performance focused:
"Optimize <module> performance with profiling and benchmarking"
"Implement caching strategy for <frequent_operation>"
"Database query optimization for <slow_endpoint>"
```

## 🎯 Multi-Skill Combinations

### **Complete Feature Development**
```bash
# Single command for full workflow:
"Design and implement <feature> with cosmo-architecture-patterns, test-driven-development, rock-test, rock-review, and prepare for rock-deploy"
```

### **Security + Performance**
```bash
# Combined focus:
"Comprehensive security and performance review for <module> with rock-review skill"
```

### **Architecture + Testing**
```bash
# Design with testing mindset:
"Design <feature> architecture using cosmo-architecture-patterns with comprehensive test strategy"
```

## 🔧 Custom Skills for Cosmo Backend

### **Cosmo-Specific Patterns**
```bash
# Repository pattern:
"Implement repository layer for <domain> following Cosmo patterns with proper testing"

# Service layer:
"Design service layer for <business_logic> with dependency injection and error handling"

# Handler layer:
"Create API handlers for <resource> with proper validation, authentication, and response formatting"
```

### **Integration Tasks**
```bash
# External service integrations:
"Build integration connector for <service_name> with proper error handling and retry logic"
"Implement webhook handler for <external_service> with signature validation"
"Create API wrapper for <third_party_service> with rate limiting"
```

## 📝 Best Practices

### **Before Using Skills:**
1. **Clear context** - Provide sufficient information about the feature
2. **Specific requirements** - List specific business requirements
3. **Constraints** - Mention performance, security, or compatibility requirements
4. **Existing codebase** - Reference existing patterns or similar features

### **During Skill Usage:**
1. **Review generated code** - Don't blindly accept
2. **Run tests** - Verify tests pass and coverage is good
3. **Check integration** - Ensure code integrates well with existing
4. **Security review** - Verify security implications

### **After Skill Usage:**
1. **Run full test suite** - Ensure no regressions
2. **Manual testing** - Verify functionality works as expected
3. **Documentation** - Update relevant documentation
4. **Code review** - Have team member review

## 🚨 Troubleshooting

### **If skill is not activated:**
- Try rephrasing with keywords from examples
- Mention skill name explicitly: "using rock-review skill"
- Provide more specific context

### **If results are not good:**
- Provide more requirements
- Reference existing similar code
- Ask for specific patterns: "following Cosmo architecture"

### **If you need custom skill:**
- Combine multiple skills in one command
- Break down complex tasks into smaller chunks
- Use sequential commands for complex workflows