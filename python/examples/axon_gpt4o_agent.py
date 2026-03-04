import os
import time
from typing import List, Dict, Any
from openai import OpenAI
from axon import AxonClient

# 1. Initialize Axon and OpenAI
axon = AxonClient()
client = OpenAI(api_key=os.environ.get("OPENAI_API_KEY"))

# 2. Define the Agent's Tools (System Prompt)
SYSTEM_PROMPT = """
You are an AI Browser Agent powered by Axon. 
Your goal is to complete the user's task by navigating the web and interacting with elements.

HOW TO USE AXON:
1. Always start by navigating to the relevant URL.
2. Use 'axon_snapshot()' to see what's on the page. You will get a COMPACT SEMANTIC VIEW.
3. Elements have REF IDs like [e1], [a1], [n1]. 
4. Use 'axon_act(ref, action, value=None)' to interact.
   - actions: click, fill, press, hover
   - for 'fill', provide the 'value' string.

RULES:
- Do not make up REF IDs. Only use what you see in the snapshot.
- If a page changes, take a new snapshot.
- When finished, summarize what you did.
"""

def axon_agent(goal: str):
    print(f"🚀 Starting Agent with goal: {goal}")
    
    # Start a session
    session = axon.create_session(profile="ai_agent")
    print(f"🆔 Session: {session.session_id}")
    
    messages = [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": goal}
    ]
    
    # Basic Agent Loop (Max 5 steps for demo)
    for i in range(5):
        print(f"\n--- Step {i+1} ---")
        
        # 1. Ask LLM for next action
        response = client.chat.completions.create(
            model="gpt-4o",
            messages=messages,
            functions=[
                {
                    "name": "axon_navigate",
                    "description": "Navigate to a URL",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "url": {"type": "string"}
                        },
                        "required": ["url"]
                    }
                },
                {
                    "name": "axon_snapshot",
                    "description": "Get semantic snapshot of the page",
                    "parameters": {"type": "object", "properties": {}}
                },
                {
                    "name": "axon_act",
                    "description": "Interact with an element",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "ref": {"type": "string"},
                            "action": {"type": "string", "enum": ["click", "fill", "press", "hover"]},
                            "value": {"type": "string"}
                        },
                        "required": ["ref", "action"]
                    }
                }
            ]
        )
        
        message = response.choices[0].message
        
        # 2. Handle Tool Call
        if message.function_call:
            func_name = message.function_call.name
            args = eval(message.function_call.arguments)
            
            print(f"🤖 Agent wants to: {func_name}({args})")
            
            if func_name == "axon_navigate":
                result = axon.navigate(args["url"])
                tool_output = f"Navigated to {args['url']}"
            elif func_name == "axon_snapshot":
                snap = axon.snapshot()
                tool_output = snap.content # The compact text representation
                # print(f"📄 Snapshot received (~{snap.token_count} tokens)")
            elif func_name == "axon_act":
                res = axon.act(args["ref"], args["action"], args.get("value", ""))
                tool_output = str(res)
            
            messages.append(message)
            messages.append({"role": "function", "name": func_name, "content": tool_output})
        else:
            print(f"🏁 Agent finished: {message.content}")
            break

    axon.delete_session(session.session_id)

if __name__ == "__main__":
    # Example: "Find the price of the latest iPhone on Amazon"
    axon_agent("Go to google.com and search for 'Axon AI Browser'")
