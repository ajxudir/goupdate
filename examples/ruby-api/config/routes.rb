Rails.application.routes.draw do
  get '/ping', to: 'health#ping'
  get '/health', to: 'health#status'
end
