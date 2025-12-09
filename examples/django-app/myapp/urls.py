from django.urls import path
from . import views

urlpatterns = [
    path('ping/', views.ping, name='ping'),
    path('health/', views.health, name='health'),
]
