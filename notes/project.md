## Task Description

Imagine a health tech startup aiming to revolutionize personal wellness through innovative technology. Your task is to build an API server that helps users track their sleep patterns and improve their sleep quality. The service will allow users to log their sleep data and track their sleep trend.

### Requirements

**Design and Implementation:** Design an API that provides the following functionalities:
- **Log Sleep Data:** Allow users to log their sleep start and end times, along with the quality of sleep.
- **View Sleep Logs:** Enable users to view their past sleep logs.

**Core Functionality:** Focus on delivering a seamless and user-friendly experience. Consider what kind of data will be handled, how users will interact with the service, and what endpoints will be necessary.

**Documentation:** Provide a README file that explains:
- The purpose and features of your service
- Instructions on how to set up and run the server
- Examples of how to interact with your API
- A brief explanation of key design decisions

**Code Quality:** Ensure your code is clean, well-organized, tested, and adheres to best practices in Go programming.

---

## Implementation Strategy

### MoSCoW Prioritization Framework

#### Must Have 📋
**Core User Data:**
- Sleep logs: start/end timestamps with timezone support and current location reporting
- Sleep quality scores (1-10 scale)
- User chronotype classification (early bird/night owl)

**Primary Goal:** Improve sleep quality through data-driven insights and anomaly detection using LLM/ML models to provide medically relevant recommendations.

> **Note:** Real-world integration with wearables (smartwatches, smartphones, smart rings) requires a flexible API design that supports both high-frequency continuous data streams and simple discrete sleep logs.

#### Should Have 🎯
- **Intelligent Features:** ML/LLM-powered anomaly detection and personalized sleep recommendations
- **Pattern Recognition:** Simple moving averages for trend analysis
- **Contextual Logging:** Factors affecting sleep quality (alcohol consumption, illness, etc.)
- **Nap Tracking:** Support for daytime sleep sessions

#### Could Have 💡
- Performance metrics and evaluation systems
- Advanced analytics and reporting features
- Data visualization endpoints

#### Won't Have ❌
- User profile: age (birth date), weight, height, sex, activity level
- User authentication and authorization systems
- Pricing and subscription management
- Advanced user management features

### Technical Considerations

- **Error Handling:** Implement RFC 9457 compliant HTTP error responses
- **Data Architecture:** Design for scalability to handle continuous health data streams
- **Research Integration:** Leverage sleep science literature for evidence-based feature development

### Research Areas
- Sleep anomaly detection patterns
- Environmental and lifestyle factors affecting sleep quality
- Medical relevance of sleep metrics and recommendations