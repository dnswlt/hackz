import streamlit as st
import os
import pandas as pd
import plotly.express as px
import sys

# Load data
@st.cache_data
def load_data(path):
    df = pd.read_csv(path, sep="\t")
    df["Year"] = df["Year"].astype(str)  # Make sure Year is string for categorical x-axis
    return df


def main():
    path = os.path.join(os.path.dirname(sys.argv[0]), "gdp_growth.csv")
    df = load_data(path)

    st.title("GDP Growth by Country Over Time")

    # Sidebar country selection
    all_countries = df["Country"].unique()
    selected_countries = st.multiselect("Select countries", sorted(all_countries), default=["Switzerland", "Germany", "United States"])

    # Filter data
    filtered_df = df[df["Country"].isin(selected_countries)]

    # Plot
    fig = px.line(
        filtered_df,
        x="Year",
        y="GDP Growth",
        color="Country",
        markers=True,
        title="GDP Growth Over Time",
    )


    # Set clean unified hover display
    fig.update_traces(
        hovertemplate="%{y:.1f}",  # only show the y-value
    )
    
    fig.update_layout(
        xaxis_title="Year", 
        yaxis_title="GDP Growth (%)", 
        hovermode="x unified",
    )

    st.plotly_chart(fig, use_container_width=True)


if __name__ == "__main__":
    main()
