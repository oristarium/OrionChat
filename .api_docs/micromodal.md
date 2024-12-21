Usage
-----

Designed to be easy to use, it does most of the heavy lifting behind the scenes and exposes a simple api to interact with the dom.

Typically modals have an overlay which cover the rest of the content. To achieve this, it is normal to put the modal code just before the closing body tag, so that the modal overlay is relative to the body and covers the whole screen.

### 1\. Add the modal markup

The following html structure is expected for the modal to work. It can of course be extended to suit your needs. As an example of the customization, see the source code of the demo modal [here](https://gist.github.com/ghosh/4f94cf497d7090359a5c9f81caf60699#file-micromodal-html).

Plain textANTLR4BashCC#CSSCoffeeScriptCMakeDartDjangoDockerEJSErlangGitGoGraphQLGroovyHTMLJavaJavaScriptJSONJSXKotlinLaTeXLessLuaMakefileMarkdownMATLABMarkupObjective-CPerlPHPPowerShell.propertiesProtocol BuffersPythonRRubySass (Sass)Sass (Scss)SchemeSQLShellSwiftSVGTSXTypeScriptWebAssemblyYAMLXML                                    `Modal Title                  Modal Content`                    

*   This is the outermost container of the modal. Its job is to toggle the display of the modal. It is important that every modal have a unique id. By default the aria-hidden will be true. Micromodal takes care of toggling the value when required.
    
*   This is the div which acts as the overlay for the modal. Notice the data-micromodal-close on it. This is a special attribute which indicates that the element that it is on should trigger the close of the modal on click. If we remove that attribute, clicking on the overlay will not close the modal anymore.
    
*   The role="dialog" attribute is used to inform assistive technologies that content within is separate from the rest of the page. Additionally, the aria-labelledby="modal-1-title" attribute points to the id of the modal title. This is to help identify the purpose of the modal.
    
*   Ensuring that all buttons have a aria-label attribute which defines the action of the button. Notice the data-micromodal-close attribute is used on the button since we want to close the modal on press.
    

### 2\. Add micromodal.js

If you included the compiled file from the CDN into your project, you will have access to a MicroModal global variable, which you can use to instantiate the module.

In cases with a modular workflow, you can directly import the module into your project.

Plain textANTLR4BashCC#CSSCoffeeScriptCMakeDartDjangoDockerEJSErlangGitGoGraphQLGroovyHTMLJavaJavaScriptJSONJSXKotlinLaTeXLessLuaMakefileMarkdownMATLABMarkupObjective-CPerlPHPPowerShell.propertiesProtocol BuffersPythonRRubySass (Sass)Sass (Scss)SchemeSQLShellSwiftSVGTSXTypeScriptWebAssemblyYAMLXML`import MicroModal from 'micromodal';  // es6 module  var MicroModal = require('micromodal'); // commonjs module`        

### 3\. Use with data attributes

Set data-micromodal-trigger="modal-1" on an element, like a button or link, on whose click you want to show the modal. The value of the attribute, in this case modal-1 should correspond to the id of the modal you want to toggle.

Then instantiate the MicroModal module, so that it takes care of all the bindings for you.

Plain textANTLR4BashCC#CSSCoffeeScriptCMakeDartDjangoDockerEJSErlangGitGoGraphQLGroovyHTMLJavaJavaScriptJSONJSXKotlinLaTeXLessLuaMakefileMarkdownMATLABMarkupObjective-CPerlPHPPowerShell.propertiesProtocol BuffersPythonRRubySass (Sass)Sass (Scss)SchemeSQLShellSwiftSVGTSXTypeScriptWebAssemblyYAMLXML`MicroModal.init();`        

Example:-

[Trigger Modal](javascript:;)

#### 3.1. Custom data attributes

You can also specify custom attributes to open and close modals. Set data-custom-open="modal-1" to any element on the page and pass it in init method as parameter of openTrigger.

The working and usage is same as data-micromodal-trigger, but with your own defined data attribute, in this case it's data-custom-open

Similarly, you can also define custom close attribute. Example:-

Plain textANTLR4BashCC#CSSCoffeeScriptCMakeDartDjangoDockerEJSErlangGitGoGraphQLGroovyHTMLJavaJavaScriptJSONJSXKotlinLaTeXLessLuaMakefileMarkdownMATLABMarkupObjective-CPerlPHPPowerShell.propertiesProtocol BuffersPythonRRubySass (Sass)Sass (Scss)SchemeSQLShellSwiftSVGTSXTypeScriptWebAssemblyYAMLXML`close`        

### 4\. Use with javascript

You can also trigger and close modals programmatically using the show and close methods on the MicroModal object. Example:-

Plain textANTLR4BashCC#CSSCoffeeScriptCMakeDartDjangoDockerEJSErlangGitGoGraphQLGroovyHTMLJavaJavaScriptJSONJSXKotlinLaTeXLessLuaMakefileMarkdownMATLABMarkupObjective-CPerlPHPPowerShell.propertiesProtocol BuffersPythonRRubySass (Sass)Sass (Scss)SchemeSQLShellSwiftSVGTSXTypeScriptWebAssemblyYAMLXML`MicroModal.show('modal-id'); // [1]  MicroModal.close('modal-id'); // [2]`        

*   The string passed to the show method must correspond to the id of the modal to be open. You can also pass in a config object in the show method and it will apply only to this modal.
    
*   The string passed to the close method must correspond to the id of the modal to be closed