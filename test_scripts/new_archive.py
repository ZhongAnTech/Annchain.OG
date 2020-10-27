#!/usr/bin/python3

import requests
import json

url = "http://localhost:8000/new_archive"
postData = {
    "public_key": "0x769153474351324",
    "signature": "0x169153474351324",
    "op_hash": "MHgzNDc1NjIzNzA1MjM2",
    "data": "ZXlKdmNDSTZJbWx1YzJWeWRDSXNJbU52Ykd4bFkzUnBiMjRpT2lKellXMXdiR1ZmWTI5c2JHVmpkR2x2YmlJc0ltOXdYMlJoZEdFaU9uc2libUZ0WlNJNkltWjFaR0Z1SWl3aVlXUmtjbVZ6Y3lJNmV5SmphWFI1SWpvaVUyaGhibWRvWVdraUxDSnliMkZrSWpvaWVIaDRJbjBzSW14dloyOGlPbnNpZFhKc0lqb2lhSFIwY0RvdkwyRXVjRzVuSW4wc0luUmxZV05vWlhKeklqcGJJbFF4SWl3aVZESWlMQ0pVTXlJc1hYMTk="
}
resp = requests.post(url, data=json.dumps(postData))
print(resp.text)
