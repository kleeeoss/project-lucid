from django.shortcuts import render
from django.http import JsonResponse
import subprocess

class User:
    objects = None

def user_view(request):
    user_id = request.GET.get('id')
    # Safe ORM query (not a raw SQL sink)
    users = User.objects.filter(id=user_id).values()
    return JsonResponse(list(users), safe=False)

def ping_view(request):
    host = request.GET.get('host', '127.0.0.1')
    # Safe subprocess execution with list arguments (shell=False)
    res = subprocess.run(['ping', '-c', '1', host], capture_output=True, shell=False)
    return JsonResponse({'status': res.returncode})
