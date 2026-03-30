class SessionsController < ApplicationController
  allow_unauthenticated_access only: %i[ new create oauth_callback oauth_failure ]
  rate_limit to: 10, within: 3.minutes, only: :create, with: -> { redirect_to new_session_path, alert: "Try again later." }

  def new
  end

  def create
    if user = User.authenticate_by(params.permit(:email_address, :password))
      start_new_session_for user
      cookies.signed.permanent[:last_auth_provider] = { value: "password", httponly: true, same_site: :lax }
      redirect_to after_authentication_url
    else
      redirect_to new_session_path, alert: "Try another email address or password."
    end
  end

  def oauth_callback
    auth = request.env["omniauth.auth"]

    unless auth.present?
      redirect_to new_session_path, alert: "Google sign-in failed. Please try again."
      return
    end

    user, created = User.from_google_oauth(auth)
    start_new_session_for(user)
    cookies.signed.permanent[:last_auth_provider] = { value: "google_oauth2", httponly: true, same_site: :lax }

    if created
      session[:oauth_setup_notice] = true
      redirect_to edit_user_path(user), notice: "Signed in with Google. Finish your profile to continue."
    else
      redirect_to dashboard_path, notice: "Signed in with Google."
    end
  rescue ActiveRecord::RecordInvalid, ArgumentError => e
    Rails.logger.warn("Google OAuth sign-in failed: #{e.class} - #{e.message}")
    redirect_to new_session_path, alert: "Could not sign in with Google. Please try again."
  end

  def oauth_failure
    message = params[:message].presence || "unknown_error"
    redirect_to new_session_path, alert: "Google sign-in failed (#{message.humanize})."
  end

  def destroy
    terminate_session
    redirect_to new_session_path, status: :see_other
  end
end
