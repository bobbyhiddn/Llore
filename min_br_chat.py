"""
A simplified chat interface for Amazon Bedrock models, designed for airgapped environments.
Only requires boto3, botocore, and standard libraries.
"""
import os
import sys
import json
from datetime import datetime
from pathlib import Path
import boto3
from botocore.exceptions import ClientError
import re

def send_message(brt, model_id, message_log, max_retries=3, retry_delay=2):
    """
    Send messages to a Bedrock model and get a response.
    """
    # Handle different model formats based on model type
    if "claude" in model_id.lower():
        # Extract system message if present
        system_messages = [msg for msg in message_log if msg["role"] == "system"]
        system_prompt = system_messages[0]["content"] if system_messages else None
        
        # Get recent conversation messages (excluding system)
        conversation_messages = [msg for msg in message_log[-10:] if msg["role"] != "system"]
        
        # Create request body for Claude
        if system_prompt:
            native_request = {
                "anthropic_version": "bedrock-2023-05-31",
                "max_tokens": 4096,
                "temperature": 0.7,
                "top_p": 0.9,
                "system": system_prompt,
                "messages": conversation_messages
            }
        else:
            native_request = {
                "anthropic_version": "bedrock-2023-05-31",
                "max_tokens": 4096,
                "temperature": 0.7,
                "top_p": 0.9,
                "messages": conversation_messages
            }

    # Convert the request to JSON
    request = json.dumps(native_request)

    # Retry logic for handling throttling
    retries = 0
    while retries <= max_retries:
        try:
            # Invoke the model with the request
            response = brt.invoke_model(modelId=model_id, body=request)

            # Decode the response body
            model_response = json.loads(response["body"].read())

            # Extract the response text
            response_text = model_response["content"][0]["text"]
            return response_text

        except ClientError as e:
            error_code = e.response.get("Error", {}).get("Code", "")
            if error_code == "ThrottlingException" and retries < max_retries:
                wait_time = retry_delay * (2 ** retries)  # Exponential backoff
                print(f"Rate limited by model. Retrying in {wait_time} seconds...")
                import time
                time.sleep(wait_time)
                retries += 1
            else:
                print(f"Error invoking Bedrock model: {e}")
                return f"Error: {str(e)}"
        except Exception as e:
            print(f"Unexpected error: {e}")
            return f"Error: {str(e)}"

    # If we exhausted retries
    return "Error: Maximum retries exceeded when calling the model."

def main():
    """Chat with Amazon Bedrock models."""
    
    # Map model choices to actual Bedrock model IDs
    model_id = "us.anthropic.claude-3-5-sonnet-20241022-v2:0"
    print(f"Using model: claude-3.5 ({model_id})")
    
    try:
        # Initialize the Bedrock client
        brt = boto3.client("bedrock-runtime")
        
        # Validate credentials with a simple test prompt
        test_prompt = {"role": "user", "content": "test"}
        test_response = send_message(brt, model_id, [test_prompt], max_retries=0)
        
        if test_response is None:
            print("AWS credentials found. Starting chat session.")
        
    except Exception as e:
        print(f"Failed to initialize Bedrock client: {e}")
        print("Please ensure your AWS credentials are properly configured.")
        sys.exit(1)
    
    # System prompt
    message_log = [
        {"role": "system", "content": "You are a helpful AI assistant using Amazon Bedrock. You have knowledge of software development and computer science. Be concise yet thorough in your responses. When sharing code, place it within code blocks."}
    ]

    # Initial greeting
    try:
        initial_message = {"role": "user", "content": "Hello! Please introduce yourself briefly."}
        message_log.append(initial_message)
        greeting = send_message(brt, model_id, message_log)
        message_log.append({"role": "assistant", "content": greeting})
        print("\nAssistant:")
        print(greeting)
    except Exception as e:
        print(f"Error getting initial greeting: {e}")
        print("Continuing with chat. Type your first message.")
    
    # Main chat loop
    last_response = ""
    print("\nType 'help' to see available commands")
    while True:
        user_input = input("\nYou (help/quit/scribe): ")
        
        if user_input.lower() == "help":
            print("\nAvailable commands:")
            print("Type your message: Chat with the AI")
            print("help: Show commands")
            print("scribe: Save last response or extract code blocks as a file or script")
            print("quit: Exit chat")
            continue
            
        if user_input.lower() == "quit":
            print("Goodbye!")
            break

        elif user_input.lower() == "scribe":
            if not last_response:
                print("No response to save yet.")
                continue
                
            save_prompt = input("Save as (python/py/bash/sh/markdown/md/text/txt): ")
            file_ext = ".txt"  # Default extension
            content_to_save = last_response
            
            if save_prompt.lower() in ["python", "py"]:
                file_ext = ".py"
                import re
                code_blocks = re.findall(r'```(?:python|py)(.*?)```', last_response, re.DOTALL)
                if code_blocks:
                    content_to_save = '\n\n'.join(block.strip() for block in code_blocks)
                else:
                    print("No Python code blocks found. Saving full response...")
            elif save_prompt.lower() in ["bash", "sh"]:
                file_ext = ".sh"
                import re
                code_blocks = re.findall(r'```(?:bash|sh)(.*?)```', last_response, re.DOTALL)
                if code_blocks:
                    content_to_save = '\n\n'.join(block.strip() for block in code_blocks)
                else:
                    print("No bash code blocks found. Saving full response...")
            elif save_prompt.lower() in ["markdown", "md"]:
                file_ext = ".md"
                
                # Step 1: Strip ANSI color codes if any
                content = last_response
                
                # Step 2: Remove explanatory text before the main Markdown content
                explanation_pattern = re.compile(r'^.*?(?=#+ |```)', re.DOTALL)  # Match until first header or code block
                explanation_match = explanation_pattern.search(content)
                if explanation_match and len(explanation_match.group(0).strip()) > 0:
                    content = content[len(explanation_match.group(0)):]
                
                # Step 3: Remove trailing questions or comments
                end_patterns = [
                    r'Would you like me to.*?$',
                    r'Does this look good.*?$',
                    r'Do you want me to.*?$',
                    r'Is there anything else.*?$',
                    r'This README provides.*?$'
                ]
                for pattern in end_patterns:
                    content = re.sub(pattern, '', content, flags=re.DOTALL | re.IGNORECASE)
                
                # Step 4: Clean up nested code blocks and ensure proper formatting
                # Find all code blocks and preserve them
                def preserve_code_blocks(match):
                    # Keep the full code block intact, including the language specifier
                    return match.group(0)
                
                # Protect code blocks from being split incorrectly
                content = re.sub(r'(```[\w]*\n.*?\n```)', preserve_code_blocks, content, flags=re.DOTALL)
                
                # Step 5: Clean trailing whitespace and ensure proper Markdown structure
                content = content.strip()
                
                # Step 6: If the entire response is wrapped in a single code block, unwrap it
                if content.startswith('```') and content.endswith('```'):
                    lines = content.split('\n')
                    if len(lines) > 2 and lines[0].strip().startswith('```') and lines[-1].strip() == '```':
                        # Extract content, preserving nested blocks
                        content = '\n'.join(lines[1:-1])
                
                content_to_save = content
            elif save_prompt.lower() in ["text", "txt"]:
                file_ext = ".txt"
                # Extract text code blocks or use the entire response
                code_blocks = re.findall(r'```(?:text|txt)(.*?)```', last_response, re.DOTALL)
                
                if code_blocks:
                    content_to_save = '\n\n'.join(block.strip() for block in code_blocks)
                    print(f"Extracted {len(code_blocks)} text blocks.")
                else:
                    print("No text code blocks found. Using full response...")
                    content_to_save = last_response
            
            file_name = input(f"Enter filename (without {file_ext} extension): ")
            with open(f"{file_name}{file_ext}", 'w', encoding='utf-8') as f:
                # Add shebang for shell scripts
                if file_ext == ".sh":
                    f.write('#!/bin/bash\n\n')
                f.write(content_to_save)
            
            # Make scripts executable
            if file_ext in [".py", ".sh"]:
                Path(f"{file_name}{file_ext}").chmod(0o755)
            
            print(f"File saved as {file_name}{file_ext}")
            continue
        
        else:
            # Regular chat message
            message_log.append({"role": "user", "content": user_input})
            print("\nGetting response...")
            
            try:
                response = send_message(brt, model_id, message_log)
                message_log.append({"role": "assistant", "content": response})
                last_response = response
                print("\nAssistant:")
                print(response)
            except Exception as e:
                print(f"Error: {e}")

if __name__ == "__main__":
    main()
