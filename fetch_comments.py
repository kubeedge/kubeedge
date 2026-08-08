import json
import subprocess
import sys

def get_prs():
    cmd = ["gh", "pr", "list", "--repo", "kubeedge/kubeedge", "--author", "Norway-02", "--state", "all", "--limit", "50", "--json", "number,title"]
    res = subprocess.run(cmd, capture_output=True, text=True)
    return json.loads(res.stdout)

def get_comments(pr_number):
    cmd = ["gh", "pr", "view", str(pr_number), "--repo", "kubeedge/kubeedge", "--json", "comments,reviews"]
    res = subprocess.run(cmd, capture_output=True, text=True)
    return json.loads(res.stdout)

prs = get_prs()
all_feedback = []

for pr in prs:
    num = pr["number"]
    data = get_comments(num)
    
    # Extract comments
    for c in data.get("comments", []):
        author = c.get("author", {}).get("login", "")
        if author and author != "Norway-02" and author != "codecov" and author != "k8s-ci-robot":
            all_feedback.append(f"PR #{num} - Comment by {author}: {c.get('body')}")
            
    # Extract reviews
    for r in data.get("reviews", []):
        author = r.get("author", {}).get("login", "")
        if author and author != "Norway-02":
            body = r.get("body", "")
            if body:
                all_feedback.append(f"PR #{num} - Review by {author}: {body}")

with open("reviewer_feedback.txt", "w") as f:
    for feedback in all_feedback:
        f.write(feedback + "\n" + "-"*80 + "\n")

print(f"Extracted {len(all_feedback)} comments/reviews.")
